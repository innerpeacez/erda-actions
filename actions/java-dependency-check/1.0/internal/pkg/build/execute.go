package build

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/erda-project/erda-actions/actions/java-dependency-check/1.0/internal/pkg/conf"
	"github.com/erda-project/erda-actions/pkg/render"
	"github.com/erda-project/erda/pkg/envconf"
	"github.com/erda-project/erda/pkg/strutil"
)

func Execute() error {
	var cfg conf.Conf
	envconf.MustLoad(&cfg)
	fmt.Fprintln(os.Stdout, "successfully loaded action config")

	if err := scan(cfg); err != nil {
		return err
	}

	return nil
}

const (
	mvnSettingsXMLFilePath = "/opt/action/mvn/settings.xml"
)

func scan(cfg conf.Conf) error {
	// 切换工作目录
	if cfg.CodeDir != "" {
		relativePath := strutil.TrimPrefixes(cfg.CodeDir, cfg.WorkDir)
		fmt.Fprintf(os.Stdout, "change workding directory to: %s\n", relativePath)
		if err := os.Chdir(cfg.CodeDir); err != nil {
			return err
		}
	}

	// render mvn settings.xml
	if err := render.RenderTemplate(filepath.Dir(mvnSettingsXMLFilePath), map[string]string{
		"NEXUS_URL":      cfg.NexusURL,
		"NEXUS_USERNAME": cfg.NexusUsername,
		"NEXUS_PASSWORD": cfg.NexusPassword,
	}); err != nil {
		return fmt.Errorf("failed to render mvn settings.xml, err: %v", err)
	}
	if cfg.Debug {
		b, err := ioutil.ReadFile(mvnSettingsXMLFilePath)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "mvn-settings.xml: %s\n", string(b))
	}

	scanCmd := exec.Command("mvn",
		"org.owasp:dependency-check-maven:check",
		"-e", "-B", "--fail-never",
		"-DautoUpdate=false",
		"-DretireJsAnalyzerEnabled=false",
		"-DnodeAnalyzerEnabled=false",
		"-DassemblyAnalyzerEnabled=false",
		"-DfailOnError=false",
		"-Dformat=ALL",
		"-DfailBuildOnAnyVulnerability=false",
		"-DdataDirectory=/opt/action/dependency-check",
		// "-DoutputDirectory="+cfg.UploadDir, // auto upload report dir for download, but this maven-plugin has bug yet: https://github.com/jeremylong/DependencyCheck/issues/1686
		"-s", mvnSettingsXMLFilePath,
	) // use `mvn org.owasp:dependency-check-maven:help -Ddetail=true -Dgoal=check` to see more options
	scanCmd.Env = append(scanCmd.Env, fmt.Sprintf("-Xmx%sm", strconv.FormatFloat(cfg.Memory-32, 'f', 0, 64)))
	fmt.Printf("executing: %v\n", scanCmd.Args)
	scanCmd.Stdout = os.Stdout
	scanCmd.Stderr = os.Stderr
	if err := scanCmd.Run(); err != nil {
		return err
	}

	// copy result to uploadDir
	if err := exec.Command("/bin/sh", "-c", "cp target/dependency-check-* "+cfg.UploadDir).Run(); err != nil {
		return fmt.Errorf("failed to copy report for download, err: %v", err)
	}

	return nil
}
