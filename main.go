// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	mmv1product "github.com/GoogleCloudPlatform/magic-modules/mmv1/api/product"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thekad/ansible-mmv1/pkg/ansible"
	"github.com/thekad/ansible-mmv1/pkg/api"
	tpl "github.com/thekad/ansible-mmv1/pkg/template"
)

const MMV1_REPO string = "https://github.com/GoogleCloudPlatform/magic-modules"
const MIN_VERSION string = "beta"

// generationWorkerCount caps concurrent module generation (templates + formatters).
func generationWorkerCount(numModules int) int {
	if numModules <= 1 {
		return 1
	}
	n := max(runtime.NumCPU(), 2)
	n = min(n, 16)
	n = min(n, numModules)
	return n
}

// moduleJob bundles a module with its per-resource generation flags so that
// worker goroutines do not need to access shared state.
type moduleJob struct {
	module     *ansible.Module
	writeCode  bool
	writeTests bool
}

func generateModule(templateData *tpl.TemplateData, job moduleJob, noFormat bool) error {
	m := job.module
	if job.writeCode {
		log.Info().Msgf("generating code for ansible module: %s", m)
		if err := templateData.GenerateCode(m); err != nil {
			return fmt.Errorf("generate code for %s: %w", m.Name, err)
		}
		if !noFormat {
			filePath := path.Join(templateData.ModuleDirectory, fmt.Sprintf("%s.py", m.Name))
			log.Info().Msgf("formatting ansible module file: %s", filePath)
			if err := formatFile(filePath, "black"); err != nil {
				return fmt.Errorf("format %s: %w", m.Name, err)
			}
		}
	}
	if job.writeTests && len(m.Resource.Mmv1.Examples) > 0 {
		log.Info().Msgf("generating tests for ansible module: %s", m)
		if err := templateData.GenerateTests(m); err != nil {
			return fmt.Errorf("generate tests for %s: %w", m.Name, err)
		}
		if !noFormat {
			dirPath := path.Join(templateData.IntegrationTestDirectory, m.Name)
			log.Info().Msgf("formatting integration tests for %s", m.Name)
			if err := formatFile(dirPath, "yamlfmt"); err != nil {
				return fmt.Errorf("format tests for %s: %w", m.Name, err)
			}
		}
	}
	return nil
}

func generateModules(templateData *tpl.TemplateData, jobs []moduleJob, noFormat bool) error {
	if len(jobs) == 0 {
		return nil
	}
	workers := generationWorkerCount(len(jobs))
	queue := make(chan moduleJob, len(jobs))
	for _, j := range jobs {
		queue <- j
	}
	close(queue)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for range workers {
		wg.Go(func() {
			for j := range queue {
				if err := generateModule(templateData, j, noFormat); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
			}
		})
	}
	wg.Wait()
	return firstErr
}

// ProductConfig holds per-product configuration with optional resource list
// and optional skip directives for code and test generation.
//
// skip-code and skip-tests accept a list of resource names to skip, or ['*']
// to skip all resources in the product:
//
//	products:
//	  - name: cloudbuildv2
//	    skip-tests: ['*']        # skip tests for every resource in this product
//	  - name: vertexai
//	    skip-tests:              # skip tests only for the listed resources
//	      - Endpoint
//	    skip-code:               # skip code only for the listed resources
//	      - SomeResource
type ProductConfig struct {
	Name      string   `mapstructure:"name"`
	Resources []string `mapstructure:"resources"`
	SkipCode  []string `mapstructure:"skip-code"`
	SkipTests []string `mapstructure:"skip-tests"`
}

// skipContains reports whether resource (case-insensitive) is covered by the
// given skip list. ['*'] means all resources are skipped.
func skipContains(skip []string, resource string) bool {
	if slices.Contains(skip, "*") {
		return true
	}
	return slices.Contains(skip, strings.ToLower(resource))
}

// mustBindPFlag binds a cobra flag to a viper key, panicking if it fails.
// Failure is only possible for programming errors (wrong flag name), so a
// panic at startup is the appropriate response.
func mustBindPFlag(viperKey, flagName string) {
	if err := viper.BindPFlag(viperKey, rootCmd.Flags().Lookup(flagName)); err != nil {
		panic(fmt.Sprintf("failed to bind flag --%s to viper key %q: %v", flagName, viperKey, err))
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ansible-mmv1",
	Short: "Generate Ansible modules from Magic Modules",
	Long: `ansible-mmv1 generates Ansible modules from Google's Magic Modules repository.
It reads product and resource definitions and generates Python modules and integration tests.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		initLogging(cmd)
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	// Set the run function here to avoid initialization cycle
	rootCmd.Run = runGenerate

	// Git flags
	rootCmd.Flags().String("git-url", MMV1_REPO, "git repository to clone")
	rootCmd.Flags().String("git-dir", "magic-modules", "path to clone magic modules repo")
	rootCmd.Flags().String("git-rev", "main", "git revision to checkout")
	rootCmd.Flags().Bool("git-pull", false, "git pull before checkout")
	rootCmd.Flags().Bool("no-git-clone", false, "skip git clone/checkout (use existing git directory)")
	rootCmd.Flags().Bool("no-git-pull", false, "negate --git-pull (useful to override config file)")

	// Path flags
	rootCmd.Flags().StringP("output", "o", "output", "path to write autogenerated files")
	rootCmd.Flags().StringP("overlay", "O", "overlay", "overlay directory (layout: products/, templates/) merged over the cloned mmv1 root")

	// Generation flags
	rootCmd.Flags().StringSlice("products", []string{}, "comma-separated list of products to generate")
	rootCmd.Flags().StringSlice("resources", []string{}, "comma-separated list of resources to generate")
	rootCmd.Flags().Bool("no-code", false, "skip code generation")
	rootCmd.Flags().Bool("no-tests", false, "skip test generation")
	rootCmd.Flags().Bool("no-format", false, "skip formatting files (i.e. black/yamlfmt)")
	rootCmd.Flags().Bool("overwrite", false, "overwrite existing files")
	rootCmd.Flags().String("min-version", MIN_VERSION, "minimum version to generate")

	// Logging flag
	rootCmd.Flags().StringP("log-level", "l", "info", "log level (trace, debug, info, warn, error, fatal)")

	// Config file flag
	rootCmd.Flags().StringP("config", "C", "config.yaml", "path to config file")

	// Bind flags to viper (only for options that can come from config file).
	mustBindPFlag("git.url", "git-url")
	mustBindPFlag("git.dir", "git-dir")
	mustBindPFlag("git.rev", "git-rev")
	mustBindPFlag("git.pull", "git-pull")
	mustBindPFlag("overlay", "overlay")
	mustBindPFlag("overwrite", "overwrite")
}

func initLogging(cmd *cobra.Command) {
	logLevelStr, _ := cmd.Flags().GetString("log-level")

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	})

	switch strings.ToLower(logLevelStr) {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Warn().Msgf("Unrecognized log level '%s', defaulting to 'info'", logLevelStr)
	}

	log.Debug().Msgf("Log level set to %s", zerolog.GlobalLevel())
}

func initConfig() {
	// Check for config file flag
	configFile, _ := rootCmd.Flags().GetString("config")
	if configFile != "" {
		viper.SetConfigFile(configFile)
	}

	// Read config file (if it exists)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error occurred
			log.Warn().Err(err).Msg("error reading config file")
		}
	} else {
		log.Info().Msgf("using config file: %s", viper.ConfigFileUsed())
	}
}

// buildProductResourceMap builds a map of product names to their resource lists
func buildProductResourceMap(products []ProductConfig) map[string][]string {
	prm := make(map[string][]string)
	for _, p := range products {
		// Normalize resource names to lowercase
		resources := make([]string, len(p.Resources))
		for i, r := range p.Resources {
			resources[i] = strings.ToLower(strings.TrimSpace(r))
		}
		prm[strings.ToLower(p.Name)] = resources
	}
	return prm
}

// getProductNames extracts product names from config
func getProductNames(products []ProductConfig) []string {
	names := make([]string, len(products))
	for i, p := range products {
		names[i] = strings.ToLower(p.Name)
	}
	return names
}

// getConfigProducts returns product entries from the config file only (viper).
// Nil or empty means the config does not define a product list, so no config-level
// product/resource filtering is applied.
func getConfigProducts() []ProductConfig {
	var products []ProductConfig
	if err := viper.UnmarshalKey("products", &products); err != nil || len(products) == 0 {
		return nil
	}
	return products
}

// getCLIProductResourceFilters returns normalized --products and --resources from the CLI.
func getCLIProductResourceFilters(cmd *cobra.Command) (products []string, resources []string) {
	productNames, _ := cmd.Flags().GetStringSlice("products")
	resourceNames, _ := cmd.Flags().GetStringSlice("resources")
	products = make([]string, 0, len(productNames))
	for _, p := range productNames {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			products = append(products, p)
		}
	}
	resources = make([]string, 0, len(resourceNames))
	for _, r := range resourceNames {
		r = strings.ToLower(strings.TrimSpace(r))
		if r != "" {
			resources = append(resources, r)
		}
	}
	return products, resources
}

// doGitClone clones the git repository to the given path
func doGitClone(path string, ref string, pull bool, url string) error {
	log.Info().Msgf("git-clone %s to %s\n", url, path)
	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	})
	if err != nil {
		log.Warn().Msgf("git-clone: %s", err)
	}

	repo, err := git.PlainOpen(path)
	if err != nil {
		return fmt.Errorf("can't open git dir %s: %v", path, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("can't open worktree: %v", err)
	}

	if pull {
		if err := worktree.Pull(&git.PullOptions{RemoteName: "origin"}); err != nil {
			log.Warn().Msgf("can't pull from 'origin' remote: %v", err)
		} else {
			log.Info().Msg("git-pull: success")
		}
	}

	h, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return fmt.Errorf("can't resolve revision %s: :%v", ref, err)
	}

	coOpts := git.CheckoutOptions{
		Hash:  *h,
		Force: true,
	}
	if err := worktree.Checkout(&coOpts); err != nil {
		return fmt.Errorf("can't checkout worktree: %v", err)
	}

	log.Info().Msgf("git-checkout %s: success", ref)

	return nil
}

func runGenerate(cmd *cobra.Command, args []string) {
	// Get configuration values from viper (config file + flags merged)
	gitURL := viper.GetString("git.url")
	gitDir := viper.GetString("git.dir")
	gitRev := viper.GetString("git.rev")
	gitPull := viper.GetBool("git.pull")
	overlayPath := viper.GetString("overlay")
	overwrite := viper.GetBool("overwrite")

	// Get flag-only values (not available in config file)
	output, _ := cmd.Flags().GetString("output")
	noCode, _ := cmd.Flags().GetBool("no-code")
	noTests, _ := cmd.Flags().GetBool("no-tests")
	noFormat, _ := cmd.Flags().GetBool("no-format")
	minVersion, _ := cmd.Flags().GetString("min-version")
	noGitClone, _ := cmd.Flags().GetBool("no-git-clone")
	noGitPull, _ := cmd.Flags().GetBool("no-git-pull")

	// Apply --no-git-pull to negate --git-pull (or config file setting)
	if noGitPull {
		gitPull = false
	}

	// Filter chain: config product names passed into LoadProducts (empty = all products),
	// then per-resource config list + CLI --products/--resources in the loop below.
	configProducts := getConfigProducts()
	configProductResources := buildProductResourceMap(configProducts)
	configProductNames := getProductNames(configProducts)
	// Index per-product skip lists by lowercase product name for O(1) lookup.
	configSkipCode := make(map[string][]string, len(configProducts))
	configSkipTests := make(map[string][]string, len(configProducts))
	for _, p := range configProducts {
		key := strings.ToLower(p.Name)
		configSkipCode[key] = p.SkipCode
		configSkipTests[key] = p.SkipTests
	}
	cliProducts, cliResources := getCLIProductResourceFilters(cmd)

	absGitDir, _ := filepath.Abs(gitDir)
	var overlayDir string
	if overlayPath != "" {
		var err error
		overlayDir, err = filepath.Abs(overlayPath)
		if err != nil {
			log.Fatal().Err(err).Msg("invalid overlay path")
		}
	}

	ansibleTemplateDir, err := filepath.Abs("templates") // ansible specific templates
	if err != nil {
		log.Fatal().Err(err).Msg("invalid ansible templates path")
	}

	if noGitClone {
		log.Info().Msg("skipping git clone/checkout (--no-git-clone)")
	} else {
		if err := doGitClone(absGitDir, gitRev, gitPull, gitURL); err != nil {
			log.Fatal().Err(err).Msg("failed to clone git repository")
		}
	}

	mmv1Root := filepath.Join(absGitDir, "mmv1")
	log.Info().Msgf("mmv1 base directory: %s", mmv1Root)
	if overlayDir == "" {
		log.Info().Msg("overlay directory: (none)")
	} else {
		log.Info().Msgf("overlay directory: %s", overlayDir)
	}

	sysfs, loader, err := api.LoadProducts(mmv1Root, overlayDir, minVersion, configProductNames)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load Magic Modules (loader)")
	}

	log.Info().Msgf("ansible specific templates: %s", ansibleTemplateDir)
	templateData := tpl.NewTemplateData(ansibleTemplateDir, output, overwrite)
	log.Debug().Msgf("template data: %v", templateData)

	jobsToRun := []moduleJob{}
	minVersionObj := &mmv1product.Version{Name: minVersion}

	for _, pKey := range api.ProductKeys(loader) {
		short := api.ShortName(pKey)
		// CLI --products (config product scope is already applied in LoadProducts).
		if len(cliProducts) > 0 && !slices.Contains(cliProducts, short) {
			continue
		}

		mmv1Product := loader.Products[pKey]
		apiProd := api.WrapProduct(mmv1Product, mmv1Root)
		configResourceList := configProductResources[short]
		log.Debug().Msgf("config resource list for %s: %v", short, configResourceList)

		for _, mmRes := range mmv1Product.Objects {
			if mmRes == nil {
				continue
			}
			resLower := strings.ToLower(mmRes.Name)
			// First filter: per-product resources from config (non-empty list only).
			if len(configResourceList) > 0 && !slices.Contains(configResourceList, resLower) {
				continue
			}
			// Second filter: CLI --resources (if any).
			if len(cliResources) > 0 && !slices.Contains(cliResources, resLower) {
				continue
			}

			if err := api.ReloadAnsibleExamples(mmRes, sysfs); err != nil {
				log.Fatal().Err(err).Str("product", short).Str("resource", mmRes.Name).Msg("failed to load Ansible example templates")
			}

			r := api.WrapResource(mmRes, apiProd, mmv1Root)
			if r.Mmv1.NotInVersion(minVersionObj) {
				log.Warn().Msgf("resource %s.%s minimum version is %v, but %s is required", r.Parent.Name, r.Name, r.MinVersion(), minVersion)
				continue
			}

			module := ansible.NewFromResource(r)
			module.MinVersion = r.MinVersion()

			// Resolve per-resource skip flags.
			// CLI --no-code / --no-tests act as a global override; config-level
			// skip-code / skip-tests refine at the product or resource level.
			writeCode := !noCode && !skipContains(configSkipCode[short], mmRes.Name)
			writeTests := !noTests && !skipContains(configSkipTests[short], mmRes.Name)
			if !writeCode {
				log.Debug().Msgf("skipping code generation for %s.%s", short, mmRes.Name)
			}
			if !writeTests {
				log.Debug().Msgf("skipping test generation for %s.%s", short, mmRes.Name)
			}

			jobsToRun = append(jobsToRun, moduleJob{
				module:     module,
				writeCode:  writeCode,
				writeTests: writeTests,
			})
		}
	}
	if err := generateModules(templateData, jobsToRun, noFormat); err != nil {
		log.Fatal().Err(err).Msg("module generation failed")
	}
}

func formatFile(filePath string, formatType string) error {
	log.Debug().Msgf("running %s on file: %s", formatType, filePath)
	switch formatType {
	case "black":
		if blackCmd := which("black"); blackCmd == "" {
			return fmt.Errorf("black not found in PATH")
		} else {
			return runCommand(fmt.Sprintf("%s --quiet %s", blackCmd, filePath), filepath.Dir(filePath))
		}
	case "yamlfmt":
		if yamlFmtCmd := which("yamlfmt"); yamlFmtCmd == "" {
			return fmt.Errorf("yamlfmt not found in PATH")
		} else {
			return runCommand(fmt.Sprintf("yamlfmt %s", filePath), filePath)
		}
	}
	return nil
}

// which searches for the given executable and returns the full path to it
// Returns empty string if the command is not found
func which(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}

	return ""
}

func runCommand(command string, dir string) error {
	parts := strings.Split(command, " ")
	cmd := exec.Command(parts[0], parts[1:]...)

	// Set working directory if provided
	if dir != "" {
		log.Debug().Msgf("changing directory to: %s", dir)
		cmd.Dir = dir
	}

	log.Debug().Msgf("running command: %s", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("command failed")
	}
}
