package docker

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

const (
	CF_PUSH_TIMEOUT                       = 4 * time.Minute
	LONG_CURL_TIMEOUT                     = 4 * time.Minute
	DOCKER_IMAGE_DOWNLOAD_DEFAULT_TIMEOUT = 10 * time.Minute
)

var context helpers.SuiteContext

func guidForAppName(appName string) string {
	cfApp := cf.Cf("app", appName, "--guid")
	Expect(cfApp.Wait()).To(Exit(0))

	appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
	Expect(appGuid).NotTo(Equal(""))
	return appGuid
}

func guidForSpaceName(spaceName string) string {
	cfSpace := cf.Cf("space", spaceName, "--guid")
	Expect(cfSpace.Wait()).To(Exit(0))

	spaceGuid := strings.TrimSpace(string(cfSpace.Out.Contents()))
	Expect(spaceGuid).NotTo(Equal(""))
	return spaceGuid
}

func createDockerApp(appName, payload string) {
	Eventually(cf.Cf("curl", "/v2/apps", "-X", "POST", "-d", payload)).Should(Exit(0))
	domain := helpers.LoadConfig().AppsDomain
	Eventually(cf.Cf("create-route", context.RegularUserContext().Space, domain, "-n", appName)).Should(Exit(0))
	Eventually(cf.Cf("map-route", appName, domain, "-n", appName)).Should(Exit(0))
}

func assertImageAvailable(registryAddress string, imageName string) {
	client := http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://%s/v1/search?q=%s", registryAddress, imageName))
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	bytes, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(bytes)).To(ContainSubstring("library/" + imageName))
}

func TestApplications(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	config := helpers.LoadConfig()
	context = helpers.NewContext(config)
	environment := helpers.NewEnvironment(context)

	BeforeSuite(func() {
		environment.Setup()
	})

	AfterSuite(func() {
		environment.Teardown()
	})

	componentName := "Diego-Docker"

	rs := []Reporter{}

	if config.ArtifactsDirectory != "" {
		helpers.EnableCFTrace(config, componentName)
		rs = append(rs, helpers.NewJUnitReporter(config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

func getAppLogs(appName string) string {
	cfLogs := cf.Cf("logs", appName, "--recent")
	Expect(cfLogs.Wait()).To(Exit(0))
	return string(cfLogs.Out.Contents())
}

func getAppImageDetails(appName string) (string, string) {
	contents := getAppLogs(appName)

	//TODO: Replace with list all droplets API (/v3/droplets)
	r := regexp.MustCompile(".*Docker image will be cached as ([0-z.:]+)/([0-z-]+)")
	imageParts := r.FindStringSubmatch(contents)
	Expect(len(imageParts)).Should(Equal(3))

	return imageParts[1], imageParts[2]
}
