package integration_test

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"

	"testing"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var agoutiDriver *agouti.WebDriver

var apparatchikCmd *exec.Cmd

var _ = BeforeSuite(func(done Done) {

	agoutiDriver = agouti.PhantomJS()
	Expect(agoutiDriver.Start()).To(Succeed())
	Expect(buildApparatchik()).To(Succeed())
	var err error
	apparatchikCmd, err = startApparatchik()
	Expect(err).ToNot(HaveOccurred())

	waitForApparatchikToStart(12080)
	close(done)
}, 5.0)

var _ = AfterSuite(func() {
	Expect(agoutiDriver.Stop()).To(Succeed())
	Expect(apparatchikCmd.Process.Kill()).To(Succeed())
	apparatchikCmd.Wait()
})

func buildApparatchik() error {
	cmd := exec.Command("go", "build", ".")
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	cmd.Dir = filepath.Dir(wd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ())
	return cmd.Run()
}

func startApparatchik() (*exec.Cmd, error) {
	cmd := exec.Command("./apparatchik", "--port", "12080")
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cmd.Dir = filepath.Dir(wd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ())
	return cmd, cmd.Start()
}

func waitForApparatchikToStart(port int) {
	for {
		response, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
		if err == nil && response.StatusCode == 200 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}
