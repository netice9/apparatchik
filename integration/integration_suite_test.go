package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
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

var _ = BeforeSuite(func(done Done) {

	agoutiDriver = agouti.ChromeDriver()
	Expect(agoutiDriver.Start()).To(Succeed())
	Expect(buildApparatchik()).To(Succeed())
	startApparatchik()
	close(done)
}, 5.0)

var _ = AfterSuite(func(done Done) {
	stopApparatchik()
	Expect(agoutiDriver.Stop()).To(Succeed())
	close(done)
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

var apparatchikCommand *exec.Cmd

func startApparatchik() {
	cmd, err := startApparatchikProcess()
	Expect(err).ToNot(HaveOccurred())
	apparatchikCommand = cmd
	waitForApparatchikToStart()
}

func stopApparatchik() {
	Expect(apparatchikCommand.Process.Signal(syscall.SIGTERM)).To(Succeed())
	apparatchikCommand.Wait()
}

func startApparatchikProcess() (*exec.Cmd, error) {
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

func waitForApparatchikToStart() {

	for {
		response, err := http.Get("http://localhost:12080/")
		if err == nil && response.StatusCode == 200 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

}

func clearApparatchik() {
	response, err := http.Get("http://localhost:12080/api/v1.0/applications")
	Expect(err).ToNot(HaveOccurred())
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	Expect(err).ToNot(HaveOccurred())
	names := []string{}
	Expect(json.Unmarshal(data, &names)).To(Succeed())

	for _, appName := range names {

		req, err := http.NewRequest("DELETE", fmt.Sprintf("http://localhost:12080/api/v1.0/applications/%s", appName), nil)
		Expect(err).ToNot(HaveOccurred())
		response, err := http.DefaultClient.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(response.StatusCode).To(Equal(204))

	}
}
