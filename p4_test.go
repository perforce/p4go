/*******************************************************************************

Copyright (c) 2024, Perforce Software, Inc.  All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1.  Redistributions of source code must retain the above copyright
    notice, this list of conditions and the following disclaimer.

2.  Redistributions in binary form must reproduce the above copyright
    notice, this list of conditions and the following disclaimer in the
    documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL PERFORCE SOFTWARE, INC. BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

*******************************************************************************/

package p4

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"math/rand"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Function to check and set environment variable
func setEnv(key, value string) error {
	return os.Setenv(key, value)
}

// Function to get the value of an environment variable
func getEnv(key string) string {
	return os.Getenv(key)
}

// Function to get the operating system name
func getOS() string {
	// You can use runtime.GOOS to check the operating system
	return os.Getenv("GOOS") // This is just an example. You can use runtime.GOOS directly if needed.
}

func isDirEmpty(path string) (bool, error) {
	files, err := filepath.Glob(filepath.Join(path, "*"))
	if err != nil {
		return false, err
	}
	return len(files) == 0, nil
}

func removeDirIfEmpty(path string) error {

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return os.Remove(path)
	}
	return nil
}

// Add fields to store the Perforce server instance or connection details
type PerforceTestSuite struct {
	suite.Suite
	TotalTests      int
	PassedTests     int
	FailedTestsList []string
	testName        string
	startDir        string
	p4Port          string
	clientRoot      string
	serverRoot      string
	testRoot        string
	suiteRoot       string
	enviroFile      string
	p4d             string
	noCleanup       bool
	p4api           *P4
}

// InitSuite initializes the suite
func (s *PerforceTestSuite) SetupSuite() {
	// Adding test counter fields
	s.TotalTests = 0
	s.PassedTests = 0
	fmt.Println("Starting test suite...")

	// Define start and root directories
	s.startDir, _ = os.Getwd()
	s.suiteRoot = s.suiteRootPath()

	s.p4d = os.Getenv("P4D_BIN")
	if s.p4d == "" {
		s.p4d = "p4d"
	}
}

// SetupSuite starts a new Perforce server before each test
func (s *PerforceTestSuite) SetupTest() {
	s.TotalTests++
	// Compute paths
	s.testRoot = s.testRootPath()
	s.clientRoot = s.clientRootPath()
	s.serverRoot = s.serverRootPath()
	s.enviroFile = s.enviroFilePath()
	s.noCleanup = false

	// Define P4 port (can be dynamic)
	s.p4Port = fmt.Sprintf("rsh:%s %s -i", s.p4d, s.p4dParams())
	// s.T().Logf("p4 port params: %s", s.p4Port)

	// Remove `P4CONFIG` and `P4ENVIRO` from the environment
	os.Unsetenv("P4CONFIG")
	os.Unsetenv("P4ENVIRO")

	// Set the `P4ENVIRO` environment variable
	err := os.Setenv("P4ENVIRO", s.enviroFile)
	s.Require().NoError(err, "Failed to set P4ENVIRO")

	// Create the workspace directory structure
	s.createWorkspaceTree()

	// Change the current working directory to `clientRoot`
	err = os.Chdir(s.clientRoot)
	s.Require().NoError(err, "Failed to chdir to clientroot")

	// Remove `PWD` from the environment
	os.Unsetenv("PWD")

	s.initClient()
	s.Require().NoError(err, "Failed to initialize client")

}

// TearDownSuite runs after the suite finishes
func (s *PerforceTestSuite) TearDownSuite() {
	s.retryRmRf(s.suiteRoot)
	err := removeDirIfEmpty(s.suiteRoot)
	if err != nil {
		fmt.Printf("Failed to remove testroot directory %s: %v\n", s.testRoot, err)
	}

	fmt.Printf("\nTest suite finished: %d/%d tests passed \n", s.PassedTests, s.TotalTests)
	if len(s.FailedTestsList) > 0 {
		fmt.Println("\nFailed tests: ")
		for n, testName := range s.FailedTestsList {
			fmt.Printf("\t   %d. %s\n", n+1, testName)
		}
	}
	fmt.Println("")
}

func (s *PerforceTestSuite) TearDownTest() {
	// Disconnect and clean up resources
	if s.p4api.Connected() {
		_, err := s.p4api.Disconnect()
		s.Require().NoError(err, "Failed to disconnect from Perforce server")
	}

	// Restore to the initial directory
	err := os.Chdir(s.startDir)
	if err != nil {
		fmt.Printf("Failed to change directory to %s: %v\n", s.startDir, err)
	}

	if !s.noCleanup {
		// Cleanup server root and client root directories
		s.retryRmRf(s.serverRoot)
		s.retryRmRf(s.clientRoot)

		// Remove the enviroFile if it exists
		if _, err := os.Stat(s.enviroFile); err == nil {
			err = os.Remove(s.enviroFile)
			if err != nil {
				fmt.Printf("Failed to remove enviro file %s: %v\n", s.enviroFile, err)
			}
		}

		// Remove test root and root directory if empty
		err = removeDirIfEmpty(s.serverRoot)
		if err != nil {
			fmt.Printf("Failed to remove server root directory %s: %v\n", s.serverRoot, err)
		}
		err = removeDirIfEmpty(s.testRoot)
		if err != nil {
			fmt.Printf("Failed to remove testroot directory %s: %v\n", s.testRoot, err)
		}
	}
	if !s.T().Failed() {
		s.PassedTests++
	} else {
		s.FailedTestsList = append(s.FailedTestsList, s.testName)
	}
}

func (s *PerforceTestSuite) retryRmRf(path string) {
	for {
		// Remove the path recursively
		err := os.RemoveAll(path)
		if err != nil {
			fmt.Printf("Failed to remove %s: %v\n", path, err)
		}

		// Check if the directory is empty
		empty, err := isDirEmpty(path)
		if err != nil {
			fmt.Printf("Error checking directory %s: %v\n", path, err)
		}
		if empty {
			break
		}

		time.Sleep(100 * time.Millisecond) // Avoid a tight loop
	}
}

func (s *PerforceTestSuite) createWorkspaceTree() {
	s.initClient()

	// Ensure serverRoot exists
	err := os.MkdirAll(s.serverRoot, 0755)
	s.Require().NoError(err, "Failed to create server root directory")

	// Ensure clientRoot exists
	err = os.MkdirAll(s.clientRoot, 0755)
	s.Require().NoError(err, "Failed to create client root directory")

	// Create P4Config and Enviro files
	err = s.createP4ConfigFile()
	s.Require().NoError(err, "Failed to create P4Config file")

	err = s.createEnviroFile()
	s.Require().NoError(err, "Failed to create Enviro file")
}

func (s *PerforceTestSuite) initClient() {
	s.p4api = New()
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	_, _ = s.p4api.SetCharset("none")

	// Check for the P4GO_TEST_PORT environment variable
	if port, exists := os.LookupEnv("P4GO_TEST_PORT"); exists {
		s.p4api.SetPort(port)
	} else {
		// Set the default port if P4GO_TEST_PORT isn't set
		s.p4api.SetPort(s.p4Port)
	}
	clientName := strings.ToLower(s.testName)
	s.p4api.SetClient(clientName)

}

// Need to add fetch client function in the source code
func (s *PerforceTestSuite) createClient() {
	// s.p4api = New()
	s.p4api.SetPort(s.p4Port)

	if !s.p4api.Connected() {
		ret, err := s.p4api.Connect()
		assert.True(s.T(), ret, "Failed to connect to Perforce server")
		assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	}

	client_specs, err := s.p4api.RunFetch("client")

	if err != nil {
		assert.Fail(s.T(), "failed to fetch client dict")
	}

	client_specs["Root"] = s.clientRoot
	_, err = s.p4api.RunSave("client", client_specs)
	assert.Nil(s.T(), err, "Failed to save client")
}

func (s *PerforceTestSuite) enableUnicode() {
	// Save the current directory
	startDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to the server root directory
	if err := os.Chdir(s.serverRoot); err != nil {
		log.Fatalf("Failed to change directory to %s: %v", s.serverRoot, err)
	}

	// Command to enable Unicode
	cmd := exec.Command("p4d", "-C1", "-L", "log", "-vserver=3", "-xi")

	// Run the command and capture the output
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("Command failed with error: %v, output: %s", err, output)
	}

	// Change back to the original directory
	if err := os.Chdir(startDir); err != nil {
		log.Fatalf("Failed to change directory back to %s: %v", startDir, err)
	}
}

// Compute the suite root path
func (s *PerforceTestSuite) suiteRootPath() string {
	return filepath.Join(s.startDir, "testroot")
}

// Compute the test root path
func (s *PerforceTestSuite) testRootPath() string {
	s.testName = s.T().Name()
	parts := strings.Split(s.testName, "/")
	if len(parts) > 1 {
		s.testName = parts[len(parts)-1] // Get only the test method name (e.g., TestP4Map02)
	}
	return filepath.Join(s.suiteRoot, s.sanitizeName(s.testName))
}

// Compute the client root path
func (s *PerforceTestSuite) clientRootPath() string {
	return filepath.Join(s.testRoot, "workspace")
}

// Compute the server root path
func (s *PerforceTestSuite) serverRootPath() string {
	return filepath.Join(s.testRoot, "server")
}

// Compute the enviro file path
func (s *PerforceTestSuite) enviroFilePath() string {
	return filepath.Join(s.testRoot, "p4enviro")
}

// Compute the P4D parameters
func (s *PerforceTestSuite) p4dParams() string {
	logParam := s.logParam()
	return fmt.Sprintf("%s -r %s -C1 -J off", logParam, s.serverRoot)
}

// Log parameter for P4D on Windows
func (s *PerforceTestSuite) logParam() string {
	if s.windowsTest() {
		return fmt.Sprintf("-L %s", filepath.Join(s.testRoot, "p4d.log"))
	}
	return ""
}

// Check if the test is running on Windows
func (s *PerforceTestSuite) windowsTest() bool {
	return runtime.GOOS == "windows" || os.Getenv("OS") == "Windows_NT" || os.Getenv("OS") == "MINGW"
}

// Create the P4Config file
func (s *PerforceTestSuite) createP4ConfigFile() error {
	// Check if the P4CONFIG environment variable is set
	p4Config, ok := os.LookupEnv("P4CONFIG")
	if !ok {
		// fmt.Println("P4CONFIG If not set, return without doing anything")
		return nil
	}

	// Build the full path for the P4CONFIG file
	p4ConfigPath := filepath.Join(s.clientRoot, p4Config)

	// Open or create the file and write the P4PORT value to it
	file, err := os.Create(p4ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create P4CONFIG file: %w", err)
	}
	defer file.Close()

	// Write the P4PORT value to the file
	_, err = file.WriteString(fmt.Sprintf("P4PORT=%s\n", s.p4Port))
	if err != nil {
		return fmt.Errorf("failed to write to P4CONFIG file: %w", err)
	}

	fmt.Println("P4CONFIG file created at:", p4ConfigPath)
	return nil
}

// Create the Enviro file
func (s *PerforceTestSuite) createEnviroFile() error {
	enviro, ok := os.LookupEnv("P4ENVIRO")

	if !ok {
		fmt.Println("P4ENVIRO If not set, return without doing anything")
		return nil
	}

	file, err := os.Create(enviro)
	if err != nil {
		return err
	}
	defer file.Close()

	return nil
}

// Sanitize a name for filesystem use
func (s *PerforceTestSuite) sanitizeName(name string) string {
	return strings.ReplaceAll(name, " ", "_")
}

func (s *PerforceTestSuite) TestInfo() {
	s.p4api = New()
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	s.p4api.SetPort(s.p4Port)
	ret, err := s.p4api.Connect()
	assert.True(s.T(), ret, "Failed to connect to Perforce server")
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")

	// Check if the client is connected
	connected := s.p4api.Connected()
	assert.True(s.T(), connected, "Connection dropped")

	info, _ := s.p4api.Run("info")
	assert.True(s.T(), len(info) > 0, "Info command failed")
	assert.IsType(s.T(), Dictionary{}, info[0], "P4#run_info didn't return a p4.Dictionary")

	infoDict := info[0].(Dictionary)
	assert.Equal(s.T(), s.serverRoot, infoDict["serverRoot"], "Incorrect server root")

	ret, err = s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestEnvironment() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")
	setSupported := !(getOS() == "windows" || getOS() == "darwin" || getOS() == "mingw")

	// Only proceed if environment variable is not already set and OS is supported
	if getEnv("P4DESCRIPTION") == "" && setSupported {
		// Set the environment variable P4DESCRIPTION to 'foo'
		err := setEnv("P4DESCRIPTION", "foo")
		require.NoError(s.T(), err, "Cannot set P4DESCRIPTION in registry")

		// Verify that the environment variable was set to 'foo'
		assert.Equal(s.T(), "foo", getEnv("P4DESCRIPTION"))

		// Clear the environment variable
		err = setEnv("P4DESCRIPTION", "")
		require.NoError(s.T(), err, "Cannot clear P4DESCRIPTION from registry")
	}

	s.p4api.SetUser("tony")
	s.p4api.SetClient("myworkstation")
	s.p4api.SetPort("myserver:1666")
	s.p4api.SetPassword("mypass")
	s.p4api.SetProg("somescript")
	s.p4api.SetVersion("2007.3/12345")

	assert.Equal(s.T(), s.p4api.User(), "tony")
	assert.Equal(s.T(), s.p4api.Client(), "myworkstation")
	assert.Equal(s.T(), s.p4api.Port(), "myserver:1666")
	assert.Equal(s.T(), s.p4api.Password(), "mypass")
	assert.Equal(s.T(), s.p4api.Prog(), "somescript")
	assert.Equal(s.T(), s.p4api.Version(), "2007.3/12345")

	s.p4api.SetEnviroFile("/tmp/enviro_file")
	assert.Equal(s.T(), s.p4api.EnviroFile(), "/tmp/enviro_file")

	assert.Equal(s.T(), s.p4api.EVar("foo"), "")
	s.p4api.SetEVar("foo", "bar")
	assert.Equal(s.T(), s.p4api.EVar("foo"), "bar")

	s.p4api.Close()
}

func (s *PerforceTestSuite) TestClient() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")
	if !s.p4api.Connected() {
		_, err := s.p4api.Connect()
		assert.Nil(s.T(), err, "Failed to connect to Perforce server")

	}

	client_specs, err := s.p4api.RunFetch("client")

	if err != nil {
		assert.Fail(s.T(), "failed to get client dict")
	}

	client_specs["Root"] = s.clientRoot
	client_specs["Description"] = "Test client... \n"

	// Add multiline view mappings as an array of strings with comment
	clientName := client_specs["Client"].(string)
	client_specs["View"] = []string{
		"//depot/... //" + clientName + "/...",
		"//depot/... //" + clientName + "/test/... ##comment check",
		"## Newline Comment",
	}

	assert.IsType(s.T(), Dictionary{}, client_specs, "Client wasn't a p4.Dictionary object")
	msg, _ := s.p4api.RunSave("client", client_specs)
	assert.Equal(s.T(), "Client "+client_specs["Client"].(string)+" saved.", msg.String(), "Failed to save client")

	client_specs, err = s.p4api.RunFetch("client", clientName)

	if err != nil {
		assert.Fail(s.T(), "failed to get client dict")
	}

	assert.Equal(s.T(), client_specs["View"].([]string)[0], "//depot/... //"+clientName+"/...", "Client view mapping 1 mismatch")
	assert.Equal(s.T(), client_specs["View"].([]string)[1], "//depot/... //"+clientName+"/test/... ##comment check", "Client view mapping 2 mismatch")
	assert.Equal(s.T(), client_specs["View"].([]string)[2], "## Newline Comment", "Client view mapping 3 mismatch")

	client_specs, err = s.p4api.RunFetch("client", clientName)

	if err != nil {
		assert.Fail(s.T(), "failed to get client dict")
	}

	client_specs2, err := s.p4api.RunFetch("client")

	if err != nil {
		assert.Fail(s.T(), "failed to get client dict")
	}

	assert.IsType(s.T(), Dictionary{}, client_specs2, "Client wasn't a p4.Dictionary object")
	assert.Equal(s.T(), client_specs["Root"], client_specs2["Root"], "Client roots differ")
	assert.Equal(s.T(), client_specs["Description"], client_specs2["Description"], "Client description differ")

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestFiles() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")
	if !s.p4api.Connected() {
		_, err := s.p4api.Connect()
		assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	}

	s.createClient()
	results, err := s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 0, "Shouldn't have any open files")

	// Create the directory "test_files"
	err = os.Mkdir("test_files", 0755)
	if err != nil {
		s.T().Fatalf("Failed to create directory: %v", err)
	}

	// Files to create
	files := []string{"foo", "bar", "baz"}

	// Loop through each file and create it
	for _, fn := range files {
		// Construct the filename path
		filepath := fmt.Sprintf("test_files/%s.txt", fn)

		// Create and write to the file
		err := os.WriteFile(filepath, []byte("This is a test file\n"), 0644)
		if err != nil {
			s.T().Fatalf("Failed to create file %s: %v", filepath, err)
		}

		// Simulate adding the file to Perforce
		_, _ = s.p4api.Run("add", filepath)
	}

	// Check if the files were created and added
	for _, fn := range files {
		filepath := fmt.Sprintf("test_files/%s.txt", fn)
		_, err := os.Stat(filepath)
		assert.False(s.T(), os.IsNotExist(err), fmt.Sprintf("File %s should exist", filepath))
	}

	results, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 3, "Unexpected number of open files")

	change_specs, err := s.p4api.RunFetch("change")

	if err != nil {
		assert.Fail(s.T(), "failed to get change dict")
	}

	assert.IsType(s.T(), Dictionary{}, change_specs, "Change form is not a spec")

	change_specs["Description"] = "Add some test files\n"
	_, err = s.p4api.RunSubmit(change_specs)
	assert.Nil(s.T(), err, "Failed to submit changes")

	// Ensure no files are open and that all files are present
	results, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened' command")
	require.Len(s.T(), results, 0, "Shouldn't have any open files")
	results, err = s.p4api.Run("files", "test_files/...")
	require.NoError(s.T(), err, "Failed to run 'files' command")
	require.Len(s.T(), results, 3, "Should have 3 files in directory")

	//  Now edit the files, and submit another revision.
	results, err = s.p4api.Run("edit", "test_files/...")
	require.NoError(s.T(), err, "Failed to run 'edit' command")
	require.Len(s.T(), results, 3)
	change_specs, err = s.p4api.RunFetch("change")

	if err != nil {
		assert.Fail(s.T(), "failed to get change dict")
	}

	assert.IsType(s.T(), Dictionary{}, change_specs, "Change form is not a spec")

	change_specs["Description"] = "Editing the test files\n"
	_, err = s.p4api.RunSubmit(change_specs)
	assert.Nil(s.T(), err, "Failed to submit changes")

	results, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened' command")
	require.Len(s.T(), results, 0, "Shouldn't have any open files")

	// Now delete the test_files
	_, _ = s.p4api.Run("delete", "test_files/...")
	change_specs, err = s.p4api.RunFetch("change")

	if err != nil {
		assert.Fail(s.T(), "failed to get change dict")
	}

	assert.IsType(s.T(), Dictionary{}, change_specs, "Change form is not a spec")

	change_specs["Description"] = "Delete the test files\n"
	_, err = s.p4api.RunSubmit(change_specs)
	assert.Nil(s.T(), err, "Failed to submit changes")

	results, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 0)

	// Now re-add the test_files
	_, _ = s.p4api.Run("sync", "test_files/...#2")
	_, _ = s.p4api.Run("add", "test_files/...")

	change_specs, err = s.p4api.RunFetch("change")

	if err != nil {
		assert.Fail(s.T(), "failed to get change dict")
	}

	assert.IsType(s.T(), Dictionary{}, change_specs, "Change form is not a spec")

	change_specs["Description"] = "Re-add the test files\n"
	_, err = s.p4api.RunSubmit(change_specs)
	assert.Nil(s.T(), err, "Failed to submit changes")

	results, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 0)
	results, err = s.p4api.Run("have")
	require.NoError(s.T(), err, "Failed to run 'have' command")
	require.Len(s.T(), results, 3)

	// Now branch the files
	_, _ = s.p4api.Run("integ", "test_files/...", "test_branch/...")
	change_specs, err = s.p4api.RunFetch("change")

	if err != nil {
		assert.Fail(s.T(), "failed to get change dict")
	}

	assert.IsType(s.T(), Dictionary{}, change_specs, "Change form is not a spec")

	change_specs["Description"] = "Branching the test files\n"
	_, err = s.p4api.RunSubmit(change_specs)
	assert.Nil(s.T(), err, "Failed to submit changes")

	results, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened' command")
	require.Len(s.T(), results, 0)

	// Now branch them again
	_, _ = s.p4api.Run("integ", "test_files/...", "test_branch2/...")
	change_specs, err = s.p4api.RunFetch("change")

	if err != nil {
		assert.Fail(s.T(), "failed to get change dict")
	}

	assert.IsType(s.T(), change_specs, Dictionary{}, "Change form is not a spec")

	change_specs["Description"] = "Branching the test files again\n"
	_, err = s.p4api.RunSubmit(change_specs)
	assert.Nil(s.T(), err, "Failed to submit changes")
	result, err := s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), result, 0)

	// Now check out 'p4 filelog'

	filelog, _ := s.p4api.RunFilelog("test_files/...")
	require.Len(s.T(), filelog, 3)

	// Test DepotFile, Revisions, and Integrations
	for _, df := range filelog {
		// Test DepotFile attributes
		assert.NotEmpty(s.T(), df.Name, "DepotFile name should not be empty")

		for _, rev := range df.Revisions {
			// Test Revision attributes
			assert.Greater(s.T(), rev.Rev, 0, "Revision number should be greater than 0")
			assert.Greater(s.T(), rev.Change, 0, "Change number should be greater than 0")
			assert.NotEmpty(s.T(), rev.Action, "Action should not be empty")
			assert.NotEmpty(s.T(), rev.Type, "Type should not be empty")
			assert.NotEmpty(s.T(), rev.User, "User should not be empty")
			assert.NotEmpty(s.T(), rev.Client, "Client should not be empty")
			assert.NotEmpty(s.T(), rev.Desc, "Description should not be empty")
			assert.False(s.T(), rev.Time.IsZero(), "Time should not be zero")
			if rev.Digest != "" {
				assert.NotEmpty(s.T(), rev.Digest, "Digest should not be empty")
			}
			if rev.FileSize != "" {
				assert.NotEmpty(s.T(), rev.FileSize, "FileSize should not be empty")
			}

			// Test Integrations
			if len(rev.Integrations) > 0 {
				for _, integ := range rev.Integrations {
					assert.NotEmpty(s.T(), integ.How, "Integration 'How' should not be empty")
					assert.NotEmpty(s.T(), integ.File, "Integration 'File' should not be empty")
					assert.GreaterOrEqual(s.T(), integ.SRev, 0, "Integration 'SRev' should be >= 0")
					assert.GreaterOrEqual(s.T(), integ.ERev, 0, "Integration 'ERev' should be >= 0")
				}
			}
		}
	}

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

type StandardResolveHandler struct {
	s *PerforceTestSuite
}

func (h *StandardResolveHandler) Resolve(md P4MergeData) P4MergeStatus {
	client := h.s.p4api.Client()
	assert.Equal(h.s.T(), "//"+client+"/test_resolve/foo", md.YourName(), "Unexpected Your name: "+md.YourName())
	assert.Equal(h.s.T(), "//depot/test_resolve/foo#2", md.TheirName(), "Unexpected Their name: "+md.TheirName())
	assert.Equal(h.s.T(), "//depot/test_resolve/foo#1", md.BaseName(), "Unexpected Base name: "+md.BaseName())
	assert.Equal(h.s.T(), 4, int(md.MergeHint()), "Unexpected merge hint: %d", int(md.MergeHint()))
	return md.MergeHint()
}

type ActionResolveHandler struct {
	s *PerforceTestSuite
}

func (h *ActionResolveHandler) Resolve(md P4MergeData) P4MergeStatus {
	if md.IsActionResolve() == true {
		assert.IsType(h.s.T(), P4MergeData{}, md, "Merge data wasn't a P4MergeData object")
		assert.Empty(h.s.T(), md.YourName(), "Unexpected Your name: "+md.YourName())
		assert.Empty(h.s.T(), md.TheirName(), "Unexpected Their name: "+md.TheirName())
		assert.Empty(h.s.T(), md.BaseName(), "Unexpected Base name: "+md.BaseName())

		assert.Equal(h.s.T(), "ignore", md.YourAction().String(), "Unexpected yours_action: "+md.YourAction().String())
		assert.Equal(h.s.T(), "branch", md.TheirAction().String(), "Unexpected their_action: "+md.TheirAction().String())
		assert.Empty(h.s.T(), md.MergeAction().String(), "Unexpected merge_action: "+md.MergeAction().String())
		assert.Equal(h.s.T(), "Branch resolve", md.ActionType().String(), "Unexpected type: "+md.ActionType().String())
		assert.Equal(h.s.T(), 4, int(md.MergeHint()), "Unexpected merge_hint: %d", int(md.MergeHint()))
		return md.MergeHint()

	} else if md.IsContentResolve() == true {
		assert.Fail(h.s.T(), "Unexpected content resolve scheduled")
	} else {
		assert.Fail(h.s.T(), "Unknown resolve type scheduled")
	}
	return 1
}

type ThirdGenResolveHandler struct {
	s *PerforceTestSuite
}

func (h *ThirdGenResolveHandler) Resolve(md P4MergeData) P4MergeStatus {
	if md.IsActionResolve() {

		assert.IsType(h.s.T(), P4MergeData{}, md, "Merge data wasn't a P4MergeData object")
		assert.Empty(h.s.T(), md.YourName(), "Unexpected Your name: "+md.YourName())
		assert.Empty(h.s.T(), md.TheirName(), "Unexpected Their name: "+md.TheirName())
		assert.Empty(h.s.T(), md.BaseName(), "Unexpected Base name: "+md.BaseName())

		assert.Equal(h.s.T(), "(text+w)", md.YourAction().String(), "Unexpected yours_action: "+md.YourAction().String())
		assert.Equal(h.s.T(), "(text+x)", md.TheirAction().String(), "Unexpected their_action: "+md.TheirAction().String())
		assert.Equal(h.s.T(), "(text+Dwx)", md.MergeAction().String(), "Unexpected merge_action: "+md.MergeAction().String())
		assert.Equal(h.s.T(), "Filetype resolve", md.ActionType().String(), "Unexpected type: "+md.ActionType().String())
		assert.Equal(h.s.T(), 2, int(md.MergeHint()), "Unexpected merge_hint: %d", int(md.MergeHint()))

		return md.MergeHint()

	} else if md.IsContentResolve() {
		client := h.s.p4api.Client()
		assert.IsType(h.s.T(), P4MergeData{}, md, "Merge data wasn't a P4MergeData object")
		assert.Equal(h.s.T(), "//"+client+"/test_resolve/bar", md.YourName(), "Unexpected Your name: "+md.YourName())
		assert.Equal(h.s.T(), "//depot/test_resolve/foo#4", md.TheirName(), "Unexpected Their name: "+md.TheirName())
		assert.Equal(h.s.T(), "//depot/test_resolve/foo#3", md.BaseName(), "Unexpected Base name: "+md.BaseName())
		assert.Equal(h.s.T(), 4, int(md.MergeHint()), "Unexpected merge_hint: %d", int(md.MergeHint()))

		// assert.Len(h.s.T(), md.Info(), 2, "Unexpected resolve information: %v", md.Info())
		// res := md.Info()

		// assert.Empty(h.s.T(), md.YourAction().String(), "Unexpected yours_action: "+md.YourAction().String())
		// assert.Empty(h.s.T(), md.TheirAction().String(), "Unexpected their_action: "+md.TheirAction().String())
		// assert.Empty(h.s.T(), md.MergeAction().String(), "Unexpected merge_action: "+md.MergeAction().String())
		// assert.Empty(h.s.T(), md.ActionType().String(), "Unexpected type: "+md.ActionType().String())

		return md.MergeHint()

	} else {
		assert.Fail(h.s.T(), "Unknown resolve type scheduled")
	}
	return 1
}

type BinaryResolveHandler struct {
	s *PerforceTestSuite
}

func (h *BinaryResolveHandler) Resolve(md P4MergeData) P4MergeStatus {
	if md.IsActionResolve() {

		assert.IsType(h.s.T(), P4MergeData{}, md, "Merge data wasn't a P4MergeData object")
		assert.Empty(h.s.T(), md.YourName(), "Unexpected Your name: "+md.YourName())
		assert.Empty(h.s.T(), md.TheirName(), "Unexpected Their name: "+md.TheirName())
		assert.Empty(h.s.T(), md.BaseName(), "Unexpected Base name: "+md.BaseName())

		assert.Equal(h.s.T(), "(binary)", md.YourAction().String(), "Unexpected yours_action: "+md.YourAction().String())
		assert.Equal(h.s.T(), "(binary+x)", md.TheirAction().String(), "Unexpected their_action: "+md.TheirAction().String())
		assert.Empty(h.s.T(), md.MergeAction().String(), "Unexpected merge_action: "+md.MergeAction().String())
		assert.Equal(h.s.T(), "Filetype resolve", md.ActionType().String(), "Unexpected type: "+md.ActionType().String())
		assert.Equal(h.s.T(), 4, int(md.MergeHint()), "Unexpected merge_hint: %d", int(md.MergeHint()))

		return md.MergeHint()

	} else if md.IsContentResolve() {
		// client := h.s.p4api.Client()
		assert.IsType(h.s.T(), P4MergeData{}, md, "Merge data wasn't a P4MergeData object")
		assert.Empty(h.s.T(), md.YourName(), "Unexpected Your name: "+md.YourName())
		assert.Empty(h.s.T(), md.TheirName(), "Unexpected Their name: "+md.TheirName())
		assert.Empty(h.s.T(), md.BaseName(), "Unexpected Base name: "+md.BaseName())
		assert.Empty(h.s.T(), md.BasePath(), "Unexpected Base name: "+md.BasePath())

		relativePath := "test_resolve/tgt.bin"
		// Get the absolute path
		absolutePath, err := filepath.Abs(relativePath)
		if err != nil {
			log.Fatalf("Failed to get absolute path: %v", err)
		}
		assert.Equal(h.s.T(), absolutePath, md.YourPath(), "Unexpected your_path:"+md.YourPath())
		assert.NotEmpty(h.s.T(), md.TheirPath(), "TheirPath not set correctly")
		assert.Equal(h.s.T(), 4, int(md.MergeHint()), "Unexpected merge_hint: %d", int(md.MergeHint()))

		// assert.Empty(h.s.T(), md.YourAction().String(), "Unexpected yours_action: "+md.YourAction().String())
		// assert.Empty(h.s.T(), md.TheirAction().String(), "Unexpected their_action: "+md.TheirAction().String())
		// assert.Empty(h.s.T(), md.MergeAction().String(), "Unexpected merge_action: "+md.MergeAction().String())
		// assert.Empty(h.s.T(), md.ActionType().String(), "Unexpected type: "+md.ActionType().String())

		return md.MergeHint()

	} else {
		assert.Fail(h.s.T(), "Unknown resolve type scheduled")
	}
	return 1
}

func (s *PerforceTestSuite) TestResolve() {
	// 0 CMS_QUIT - user wants to quit
	// 1 CMS_SKIP - skip the integration record
	// 2 CMS_MERGED - accepted merged theirs and yours
	// 3 CMS_EDIT - accepted edited merge
	// 4 CMS_THEIRS - accepted theirs
	// 5 CMS_YOURS - accepted yours,

	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	s.createClient()
	testDir := "test_resolve"
	err = os.Mkdir(testDir, 0755)
	require.NoError(s.T(), err, "Failed to create directory")

	file := "foo"
	bar := "bar"
	fname := "test_resolve/" + file
	bname := "test_resolve/" + bar

	// Create the file to test resolve
	err = os.WriteFile(fname, []byte("First Line!"), 0644)
	require.NoError(s.T(), err, "Failed to create file")
	_, _ = s.p4api.Run("add", fname)
	results, err := s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 1, "Unexpected number of open files")
	changeSpec, err := s.p4api.RunFetch("change")
	require.NoError(s.T(), err, "Failed to fetch change")
	changeSpec["Description"] = "First resolve submit"
	_, err = s.p4api.RunSubmit(changeSpec)
	assert.ErrorIs(s.T(), nil, err, "Failed to submit change")
	openedFiles, err := s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to fetch opened files")
	require.Len(s.T(), openedFiles, 0, "Unexpected number of open files")

	// Create a second revision of the file
	_, _ = s.p4api.Run("edit", fname)
	results, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 1, "Unexpected number of open files")
	err = os.WriteFile(fname, []byte("Second Line."), 0644)
	require.NoError(s.T(), err, "Failed to write to file")
	changeSpec, err = s.p4api.RunFetch("change")
	require.NoError(s.T(), err, "Failed to fetch change")
	changeSpec["Description"] = "Second resolve submit"
	_, err = s.p4api.RunSubmit(changeSpec)
	assert.ErrorIs(s.T(), nil, err, "Failed to submit change")
	openedFiles, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to fetch opened files")
	require.Len(s.T(), openedFiles, 0, "Unexpected number of open files")

	// Now sync to rev #1
	_, _ = s.p4api.Run("sync", fname+"#1")

	// open the file for edit and sync to schedule a resolve
	_, _ = s.p4api.Run("edit", fname)
	op, _ := s.p4api.Run("sync")
	assert.Contains(s.T(), op[1].(P4Message).lines[0].fmt, "must resolve")

	// ...and test a standard resolve
	handler := &StandardResolveHandler{s: s}

	// Call SetResolveHandler with the instance
	s.p4api.SetResolveHandler(handler)
	res, _ := s.p4api.Run("resolve")
	require.Len(s.T(), res, 3, "Unexpected number of resolve messages")

	changeSpec, err = s.p4api.RunFetch("change")
	require.NoError(s.T(), err, "Failed to fetch change")
	changeSpec["Description"] = "Third resolve submit"
	_, err = s.p4api.RunSubmit(changeSpec)
	// fmt.Println(o)
	assert.ErrorIs(s.T(), nil, err, "Failed to submit change")
	results, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 0, "Unexpected number of open files")
	_, _ = s.p4api.Run("revert", "//...")

	level, _ := s.p4api.ServerLevel()
	if level >= 31 {
		//
		// Action resolve test
		//

		// Schedule a branch resolve
		assert.NotPanics(s.T(), func() {
			_, _ = s.p4api.Run("integ", "-Rb", "//depot/test_resolve/foo", "//depot/test_resolve/bar")
		}, "Problem scheduling branch resolve from 'foo' to 'bar'")

		actionResolveCount, _ := s.p4api.Run("resolve", "-n")
		assert.NotEmpty(s.T(), actionResolveCount[0].(Dictionary)["clientFile"], "Unexpected number of resolves scheduled")

		handler := &ActionResolveHandler{s: s}
		s.p4api.SetResolveHandler(handler)
		res, _ := s.p4api.Run("resolve")
		info := res[0].(Dictionary)
		assert.Equal(s.T(), "//depot/test_resolve/foo", info["fromFile"], "Unexpected fromFile in info: "+fmt.Sprintf("%v", info["fromFile"]))
		assert.Equal(s.T(), "branch", info["resolveType"], "Unexpected resolveType in info: "+fmt.Sprintf("%v", info["resolveType"]))

		changeSpec, err = s.p4api.RunFetch("change")
		require.NoError(s.T(), err, "Failed to fetch change")
		changeSpec["Description"] = "Fourth resolve submit"
		_, err = s.p4api.RunSubmit(changeSpec)
		assert.ErrorIs(s.T(), nil, err, "Failed to submit change")
		results, err := s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to run 'opened'")
		require.Len(s.T(), results, 0, "Unexpected number of open files")
		haveResults, err := s.p4api.Run("have")
		require.NoError(s.T(), err, "Failed to run 'have'")
		require.Len(s.T(), haveResults, 2, "Unexpected number of resolves scheduled")

		// Schedule a content and filetype resolve
		_, _ = s.p4api.Run("edit", "-t+x", fname)
		err = os.WriteFile(fname, []byte("Third Line."), 0644)
		require.NoError(s.T(), err, "Failed to write to file")
		_, _ = s.p4api.Run("edit", "-t+w", bname)
		results, err = s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to run 'opened'")
		require.Len(s.T(), results, 2, "Unexpected number of open files")

		changeSpec, err = s.p4api.RunFetch("change")
		require.NoError(s.T(), err, "Failed to fetch change")
		changeSpec["Description"] = "Fifth resolve submit"
		_, err = s.p4api.RunSubmit(changeSpec)
		assert.ErrorIs(s.T(), nil, err, "Failed to submit change")
		results, err = s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to run 'opened'")
		require.Len(s.T(), results, 0, "Unexpected number of open files")

		// Force integration with 3rd generation integration engine
		_, _ = s.p4api.Run("integ", "-3", "//depot/"+fname, "//depot/"+bname)
		actionResolveCount, _ = s.p4api.Run("resolve", "-n")
		require.Len(s.T(), actionResolveCount, 2, "Unexpected number of resolves scheduled")

		thandler := &ThirdGenResolveHandler{s: s}
		s.p4api.SetResolveHandler(thandler)
		_, _ = s.p4api.Run("resolve")

		changeSpec, err = s.p4api.RunFetch("change")
		require.NoError(s.T(), err, "Failed to fetch change")
		changeSpec["Description"] = "Sixth resolve submit"
		_, err = s.p4api.RunSubmit(changeSpec)
		assert.ErrorIs(s.T(), nil, err, "Failed to submit change")
		results, err = s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to run 'opened'")
		require.Len(s.T(), results, 0, "Unexpected number of open files")

		//
		// Test binary resolves
		//

		srcbin := "src.bin"
		tgtbin := "tgt.bin"
		srcname := filepath.Join(testDir, srcbin)
		tgtname := filepath.Join(testDir, tgtbin)
		srcname = strings.ReplaceAll(srcname, "\\", "/")
		tgtname = strings.ReplaceAll(tgtname, "\\", "/")

		err = os.WriteFile(srcname, []byte("First line in binary file!"), 0644)
		require.NoError(s.T(), err, "Failed to write to file")
		_, _ = s.p4api.Run("add", "-t", "binary", srcname)

		changeSpec, err = s.p4api.RunFetch("change")
		require.NoError(s.T(), err, "Failed to fetch change")
		changeSpec["Description"] = "First binary resolve submit"
		_, err = s.p4api.RunSubmit(changeSpec)
		assert.ErrorIs(s.T(), nil, err, "Failed to submit change")
		results, err = s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to run 'opened'")
		require.Len(s.T(), results, 0, "Unexpected number of open files")

		// Branch the binary file
		assert.NotPanics(s.T(), func() {
			_, _ = s.p4api.Run("integ", "//depot/"+srcname, "//depot/"+tgtname)
		}, "Problem scheduling branch resolve from 'src.bin' to 'tgt.bin")

		changeSpec, err = s.p4api.RunFetch("change")
		require.NoError(s.T(), err, "Failed to fetch change")
		changeSpec["Description"] = "Second binary resolve submit"
		_, err = s.p4api.RunSubmit(changeSpec)
		assert.ErrorIs(s.T(), nil, err, "Failed to submit change")
		results, err = s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to run 'opened'")
		require.Len(s.T(), results, 0, "Unexpected number of open files")

		// Edit the binary file
		_, _ = s.p4api.Run("edit", "-t", "+x", srcname)
		err = os.WriteFile(srcname, []byte("Second line in binary file!"), 0644)
		require.NoError(s.T(), err, "Failed to write to file")
		results, err = s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to run 'opened'")
		assert.Len(s.T(), results, 1, "Unexpected number of open files")

		changeSpec, err = s.p4api.RunFetch("change")
		require.NoError(s.T(), err, "Failed to fetch change")
		changeSpec["Description"] = "Third binary resolve submit"
		_, err = s.p4api.RunSubmit(changeSpec)
		assert.ErrorIs(s.T(), nil, err, "Failed to submit change")
		results, err = s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to run 'opened'")
		require.Len(s.T(), results, 0, "Unexpected number of open files")

		// Integrate the binary file
		assert.NotPanics(s.T(), func() {
			level, _ := s.p4api.ServerLevel()
			if level >= 31 {
				_, _ = s.p4api.Run("integ", "-3", "//depot/"+srcname, "//depot/"+tgtname)
			} else {
				_, _ = s.p4api.Run("integ", "//depot/"+srcname, "//depot/"+tgtname)
			}
		}, "Problem integrating from 'src.bin' to 'tgt.bin")

		results, err = s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to run 'opened'")
		assert.Len(s.T(), results, 1, "Unexpected number of open files")
		level, _ := s.p4api.ServerLevel()
		if level >= 31 {
			results, err := s.p4api.Run("resolve", "-n")
			require.NoError(s.T(), err, "Failed to run 'resolve -n'")
			assert.Len(s.T(), results, 2, "Unexpected number of resolves scheduled")
		} else {
			results, err := s.p4api.Run("resolve", "-n")
			require.NoError(s.T(), err, "Failed to run 'resolve -n'")
			assert.Len(s.T(), results, 1, "Unexpected number of resolves scheduled")
		}

		bhandler := &BinaryResolveHandler{s: s}
		s.p4api.SetResolveHandler(bhandler)
		_, _ = s.p4api.Run("resolve")

		changeSpec, err = s.p4api.RunFetch("change")
		require.NoError(s.T(), err, "Failed to fetch change")
		changeSpec["Description"] = "Final binary resolve submit"
		_, err = s.p4api.RunSubmit(changeSpec)
		assert.ErrorIs(s.T(), nil, err, "Failed to submit change")
		results, err = s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to run 'opened'")
		require.Len(s.T(), results, 0, "Unexpected number of open files")

	}

	results, _ = s.p4api.Run("opened")
	if len(results) > 0 {
		_, _ = s.p4api.Run("revert", "//...")
	}

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestPassword() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")
	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	ticketFile := s.clientRoot + "/.p4tickets"
	s.p4api.SetTicketFile(ticketFile)
	assert.Equal(s.T(), s.p4api.TicketFile(), ticketFile, "Ticket file not set correctly")
	_, err = s.p4api.Run("tickets")
	assert.Nil(s.T(), err, "Failed to run 'tickets' command")

	if !s.p4api.Connected() {
		_, err := s.p4api.Connect()
		assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	}
	s.p4api.SetUser("test_user")
	userSpecs, _ := s.p4api.RunFetch("user", "test_user")
	msg, _ := s.p4api.RunPassword("", "foo")
	assert.Equal(s.T(), "Password updated.", msg.String(), "Password not updated")
	s.p4api.SetPassword("foo")
	assert.Equal(s.T(), s.p4api.Password(), "foo", "Password not set correctly")
	res, _ := s.p4api.RunLogin()
	assert.IsType(s.T(), Dictionary{}, res, "Login failed")
	//  Note: p4.password is set by the login above to the ticket.
	msg, _ = s.p4api.RunPassword("foo", "")
	assert.Equal(s.T(), "Password deleted.", msg.String(), "Password not updated")

	oldUser := userSpecs["User"]

	// Ensure that ticket file is correctly parsed for user names that contain a ':'.
	s.p4api.SetUser("foo:bar")
	userSpecs, _ = s.p4api.RunFetch("user", "foo:bar")
	_, err = s.p4api.RunSave("user", userSpecs)
	assert.Nil(s.T(), err, "Failed to save user spec")

	msg, _ = s.p4api.RunPassword("", "foo")
	assert.Equal(s.T(), "Password updated.", msg.String(), "Password not updated")

	s.p4api.SetPassword("foo")
	assert.Equal(s.T(), s.p4api.Password(), "foo", "Password not set correctly")

	res, _ = s.p4api.RunLogin()
	assert.IsType(s.T(), Dictionary{}, res, "Login failed")

	//  Note: p4.password is set by the login above to the ticket.
	msg, _ = s.p4api.RunPassword("foo", "")
	assert.Equal(s.T(), "Password deleted.", msg.String(), "Password not updated")

	_, err = os.Stat(s.p4api.TicketFile())
	require.False(s.T(), os.IsNotExist(err), "Ticket file not created")

	s.p4api.SetUser(oldUser.(string))

	allticket, _ := s.p4api.RunTickets()
	require.Len(s.T(), allticket, 2, "Unexpected number of tickets found.")

	for _, ticket := range allticket {
		assert.IsType(s.T(), map[string]string{}, ticket, "Ticket entry is not a map object")
		assert.NotNil(s.T(), ticket["Host"], "Host field not set in ticket.")
		assert.NotNil(s.T(), ticket["User"], "User field not set in ticket.")
		assert.NotNil(s.T(), ticket["Ticket"], "Ticket field not set in ticket.")
	}

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestMap() {
	clientMap := NewMap()
	defer clientMap.Close()

	assert.Empty(s.T(), clientMap.Array())

	clientMap.Insert("//depot/main/...", "//ws/main/...", P4MAP_INCLUDE)
	assert.NotEmpty(s.T(), clientMap)
	require.Len(s.T(), clientMap.Array(), 1)

	clientMap.Insert("//depot/live/...", "//ws/live/...", P4MAP_INCLUDE)
	require.Len(s.T(), clientMap.Array(), 2)

	clientMap.Insert("//depot/bad/...", "//ws/live/bad/...", P4MAP_INCLUDE)
	require.Len(s.T(), clientMap.Array(), 4)

	// Basic translation
	assert.Equal(s.T(), "//depot/main/foo/bar", clientMap.Translate("//ws/main/foo/bar", 1), "Translate (success)")
	assert.Equal(s.T(), "//ws/main/foo/bar", clientMap.Translate("//depot/main/foo/bar", 0), "Translate (success)")

	// Map joining. Create another map, and join it to the first
	wsMap := NewMap()
	defer wsMap.Close()
	assert.Empty(s.T(), wsMap.Array())
	wsMap.Insert("//ws/...", "/home/user/ws/...", P4MAP_INCLUDE)
	assert.NotEmpty(s.T(), wsMap.Array())
	require.Len(s.T(), wsMap.Array(), 1)

	rootMap := JoinMap(clientMap, wsMap)
	assert.NotEmpty(s.T(), rootMap.Array())

	// Now translate a depot path to a local path
	p := rootMap.Translate("//depot/main/foo/bar", 0)
	assert.Equal(s.T(), "/home/user/ws/main/foo/bar", p)

	// Now reverse the mappings and try again
	rootMap.Reverse()
	p = rootMap.Translate("/home/user/ws/main/foo/bar", 0)
	assert.Equal(s.T(), "//depot/main/foo/bar", p)

	// Check space handling in mappings. Insert using both methods. With,
	// and without quotes.
	spaceMap := NewMap()
	defer spaceMap.Close()
	spaceMap.Insert("//depot/space dir1/...", "//ws/space 1/...", P4MAP_INCLUDE)
	spaceMap.Insert("//depot/space dir2/...", "//ws/space 2/...", P4MAP_INCLUDE)
	spaceMap.Insert("//depot/space dir3/...", "//ws/space 3/...", P4MAP_INCLUDE)

	// Test the results using translation.
	p = spaceMap.Translate("//depot/space dir1/foo", 0)
	assert.Equal(s.T(), "//ws/space 1/foo", p)

	p = spaceMap.Translate("//depot/space dir2/foo", 0)
	assert.Equal(s.T(), "//ws/space 2/foo", p)

	p = spaceMap.Translate("//depot/space dir3/foo", 0)
	assert.Equal(s.T(), "//ws/space 3/foo", p)
}

func (s *PerforceTestSuite) TestSpecs() {

	JOBSPEC :=
		`Job:	new
Status:	open
User:	super-user
Description:
	<enter description here>
`

	BRANCHSPEC :=
		`Branch:	test_branch
Update:	2011/01/01 00:00:00
Access:	2012/01/01 00:00:00
Owner:	test
Description: Created by test.
Options:	unlocked
View:     //depot/main/... //depot/rel/...
          //depot/main/... //depot/rel2/...  ## With Comments
		  ## New Comment line`

	CHANGESPEC :=
		`Change:	new
Client:	jmistry_mac_p4go
Date:	2012/01/01 00:00:00
User:	jmistry
Status:	new
Type:	restricted
JobStatus:	open
Jobs:	job000001
		job000002
Description:
	  <enter description here>
Files: //depot/filea
	  //depot/fileb
	  ## New Comment line
`

	CLIENTSPEC :=
		`Client:	test_client
Update:	2011/01/01 00:00:00
Access:	2012/01/01 00:00:00
Owner:	test
Host:	test_host
Description:
	  Created by test.
Root:	/test/path
AltRoots:
	  /alt/path
Options:	noallwrite noclobber nocompress unlocked nomodtime normdir
SubmitOptions:	submitunchanged
Stream:	//stream/main
StreamAtChange:	@200
LineEnd:	local
ServerID:	testid
View:
	  //depot/... //test_client/... ## Inline Comment
	  ## New Comment line`

	DEPOTSPEC :=
		`Depot:	spec
Owner:	test
Date:	2012/01/01 00:00:00
Description:
	  Created by test.
Type:	local
Address:	local
Suffix:	.p4s
Map:	spec/...
SpecMap:	//spec/... ## Inline Comment
	  ## New Comment line
	  -//spec/client/...`

	GROUPSPEC :=
		`Group:	test_group
MaxResults:	unset
MaxScanRows:	unset
MaxLockTime:	unset
Timeout:	43200
PasswordTimeout:	unset
Subgroups:	test_subgroup
Owners:	test_owner
Users:	test_user`

	LABELSPEC :=
		`Label:	label1
Update:	2005/03/07 09:51:29
Access:	2012/01/18 11:11:29
Owner:	jkao
Description:
	  Created by jkao.
Options:	unlocked noautoreload
Revision:	@1234
View:
	  //depot/... ## Inline Comment
	  ## New Comment line`

	PROTECTSPEC :=
		`Protections:
	  write user * * //...
	  super user test * //...`

	STREAMSPEC :=
		`Stream:	//stream/main
Update:	2011/01/01 00:00:00
Access:	2012/01/01 00:00:00
Owner:	test
Name:	main
Parent:	none
Type:	mainline
Description:
	  Created by test.
Options:	allsubmit unlocked toparent fromparent
Paths:
	  share ...
Remapped:
	  test/...	test_remap/...
Ignored:
	  excluded/...
View:
	  //stream/main/... ...
	  //stream/main/test/... test_remap/... ## Inline Comment
	  -//stream/...excluded/... ...excluded/...
	  ## New Comment line`

	TRIGGERSPEC :=
		`Triggers:
        name form-out change "/some/script"`

	TYPEMAPSPEC :=
		`TypeMap:
	  +l //....docx`

	USERSPEC :=
		`User:	test
Type:	standard
Email:	test@test
Update:	2011/01/01 00:00:00
Access:	2012/01/01 00:00:00
FullName:	test
JobView:	status=open
Password:	testpass
Reviews:	//depot/path/...
`
	SERVERSPEC :=
		`ServerID:	418777F0-F7A5-439B-BAFC-E790E7286188
Type:	server
Name:	Maser
Address:	1666
Services:	standard
Description:
	  Created by test.`

	LDAPSPEC :=
		`Name:     myLDAPconfig
Host:     openldap.example.com
Port:     389
Encryption:    tls
BindMethod:    search
Options: nodowncase nogetattrs norealminusername
SimplePattern: someuserid
SearchBaseDN:  ou=employees,dc=example,dc=com
SearchFilter:  (cn=%user%)
SearchScope:    subtree
GroupSearchScope:  subtree`

	REMOTESPEC :=
		`RemoteID:	 test_remote
Address:	 localhost:1666
Description:	 Test remote created.
DepotMap:	 //... //remote/test_depot/...`

	REPOSPEC :=
		`Repo:	//GD1/pizza
Owner:	test
Created:	2024/12/18 14:57:16
Description: Created by test.
Options:	lfs
GconnMirrorStatus:	disabled
GconnMirrorHideFetchUrl:	false`

	HOTFILESSPEC :=
		`HotFiles:
  //depot/project1/...
  //depot/project2/... text`

	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	// Job
	job, err := s.p4api.ParseSpec("job", JOBSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse job spec: %v", err)
	}

	assert.Equal(s.T(), job["Job"], "new", "Key 'Job' has unexpected value in job P4::Spec")
	assert.Equal(s.T(), job["Status"], "open", "Key 'Status' has unexpected value in job P4::Spec")
	assert.Equal(s.T(), job["User"], "super-user", "Key 'User' has unexpected value in job P4::Spec")
	assert.NotEmpty(s.T(), job["Description"], "Key 'Description' has unexpected value in job P4::Spec")

	// Branch
	branch, err := s.p4api.ParseSpec("branch", BRANCHSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse branch spec: %v", err)
	}
	fbranch, err := s.p4api.FormatSpec("branch", branch)
	if err != nil {
		s.T().Fatalf("Failed to format branch spec: %v", err)
	}
	branch, err = s.p4api.ParseSpec("branch", fbranch)
	if err != nil {
		s.T().Fatalf("Failed to parse branch spec: %v", err)
	}
	assert.Equal(s.T(), branch["Branch"], "test_branch", "Key 'Branch' has unexpected value in branch P4::Spec")
	assert.Equal(s.T(), branch["Update"], "2011/01/01 00:00:00", "Key 'Update' has unexpected value in branch P4::Spec")
	assert.Equal(s.T(), branch["Access"], "2012/01/01 00:00:00", "Key 'Access' has unexpected value in branch P4::Spec")
	assert.Equal(s.T(), branch["Owner"], "test", "Key 'Owner' has unexpected value in branch P4::Spec")
	assert.NotEmpty(s.T(), branch["Description"], "Created by test.", "Key 'Description' has unexpected value in branch P4::Spec")
	assert.Equal(s.T(), branch["Options"], "unlocked", "Key 'Options' has unexpected value in branch P4::Spec")
	assert.Equal(s.T(), branch["View"], []string{"//depot/main/... //depot/rel/...", "//depot/main/... //depot/rel2/... ## With Comments", "## New Comment line"}, "Key 'View' has unexpected value in branch P4::Spec")

	// Change
	change, err := s.p4api.ParseSpec("change", CHANGESPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse change spec: %v", err)
	}

	assert.Equal(s.T(), change["Change"], "new", "Key 'Change' has unexpected value in change P4::Spec")
	assert.Equal(s.T(), change["Date"], "2012/01/01 00:00:00", "Key 'Date' has unexpected value in change P4::Spec")
	assert.Equal(s.T(), change["Client"], "jmistry_mac_p4go", "Key 'Client' has unexpected value in change P4::Spec")
	assert.Equal(s.T(), change["User"], "jmistry", "Key 'User' has unexpected value in change P4::Spec")
	assert.Equal(s.T(), change["Status"], "new", "Key 'Status' has unexpected value in change P4::Spec")
	assert.Equal(s.T(), change["Type"], "restricted", "Key 'Type' has unexpected value in change P4::Spec")
	assert.Equal(s.T(), change["JobStatus"], "open", "Key 'JobStatus' has unexpected value in change P4::Spec")
	assert.NotEmpty(s.T(), change["Description"], "Key 'Description' has unexpected value in change P4::Spec")
	assert.Equal(s.T(), change["Jobs"], []string{"job000001", "job000002"}, "Key 'Jobs' has unexpected value in change P4::Spec")
	assert.Equal(s.T(), change["Files"], []string{"//depot/filea", "//depot/fileb", "## New Comment line"}, "Key 'Files' has unexpected value in change P4::Spec")

	// Client
	client, err := s.p4api.ParseSpec("client", CLIENTSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse client spec: %v", err)
	}

	assert.Equal(s.T(), client["Client"], "test_client", "Key 'Client' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["Update"], "2011/01/01 00:00:00", "Key 'Update' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["Access"], "2012/01/01 00:00:00", "Key 'Access' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["Owner"], "test", "Key 'Owner' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["Host"], "test_host", "Key 'Host' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["Root"], "/test/path", "Key 'Root' has unexpected value in client P4::Spec")
	assert.NotEmpty(s.T(), client["Description"], "Key 'Description' missing from client P4::Spec")
	assert.Equal(s.T(), client["AltRoots"], []string{"/alt/path"}, "Key 'AltRoots' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["Options"], "noallwrite noclobber nocompress unlocked nomodtime normdir", "Key 'Options' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["SubmitOptions"], "submitunchanged", "Key 'SubmitOptions' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["LineEnd"], "local", "Key 'LineEnd' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["Stream"], "//stream/main", "Key 'Stream' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["StreamAtChange"], "@200", "Key 'StreamAtChange' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["ServerID"], "testid", "Key 'ServerID' has unexpected value in client P4::Spec")
	assert.Equal(s.T(), client["View"], []string{"//depot/... //test_client/... ## Inline Comment", "## New Comment line"}, "Key 'View' has unexpected value in client P4::Spec")

	// Depot
	depot, err := s.p4api.ParseSpec("depot", DEPOTSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse depot spec: %v", err)
	}

	assert.Equal(s.T(), depot["Depot"], "spec", "Key 'Depot' has unexpected value in depot P4::Spec")
	assert.Equal(s.T(), depot["Owner"], "test", "Key 'Owner' has unexpected value in depot P4::Spec")
	assert.Equal(s.T(), depot["Date"], "2012/01/01 00:00:00", "Key 'Date' has unexpected value in depot P4::Spec")
	assert.NotEmpty(s.T(), depot["Description"], "Key 'Description' missing from depot P4::Spec")
	assert.Equal(s.T(), depot["Type"], "local", "Key 'Type' has unexpected value in depot P4::Spec")
	assert.Equal(s.T(), depot["Address"], "local", "Key 'Address' has unexpected value in depot P4::Spec")
	assert.Equal(s.T(), depot["Suffix"], ".p4s", "Key 'Suffix' has unexpected value in depot P4::Spec")
	assert.Equal(s.T(), depot["Map"], "spec/...", "Key 'Map' has unexpected value in depot P4::Spec")
	assert.Equal(s.T(), depot["SpecMap"], []string{"//spec/... ## Inline Comment", "## New Comment line", "-//spec/client/..."}, "Key 'SpecMap' has unexpected value in depot P4::Spec")

	// Group
	group, err := s.p4api.ParseSpec("group", GROUPSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse group spec: %v", err)
	}

	assert.Equal(s.T(), group["Group"], "test_group", "Key 'Group' has unexpected value in group P4::Spec")
	assert.Equal(s.T(), group["MaxResults"], "unset", "Key 'MaxResults' has unexpected value in group P4::Spec")
	assert.Equal(s.T(), group["MaxScanRows"], "unset", "Key 'MaxScanRows' has unexpected value in group P4::Spec")
	assert.Equal(s.T(), group["MaxLockTime"], "unset", "Key 'MaxLockTime' has unexpected value in group P4::Spec")
	assert.Equal(s.T(), group["Timeout"], "43200", "Key 'Timeout' has unexpected value in group P4::Spec")
	assert.Equal(s.T(), group["PasswordTimeout"], "unset", "Key 'PasswordTimeout' has unexpected value in group P4::Spec")
	assert.Equal(s.T(), group["Subgroups"], []string{"test_subgroup"}, "Key 'Subgroups' has unexpected value in group P4::Spec")
	assert.Equal(s.T(), group["Owners"], []string{"test_owner"}, "Key 'Owners' has unexpected value in group P4::Spec")
	assert.Equal(s.T(), group["Users"], []string{"test_user"}, "Key 'Users' has unexpected value in group P4::Spec")

	// Label
	label, err := s.p4api.ParseSpec("label", LABELSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse label spec: %v", err)
	}

	assert.Equal(s.T(), label["Label"], "label1", "Key 'Label' has unexpected value in label P4::Spec")
	assert.Equal(s.T(), label["Update"], "2005/03/07 09:51:29", "Key 'Update' has unexpected value in label P4::Spec")
	assert.Equal(s.T(), label["Access"], "2012/01/18 11:11:29", "Key 'Access' has unexpected value in label P4::Spec")
	assert.Equal(s.T(), label["Owner"], "jkao", "Key 'Owner' has unexpected value in label P4::Spec")
	assert.NotEmpty(s.T(), label["Description"], "Key 'Description' missing from label P4::Spec")
	assert.Equal(s.T(), label["Options"], "unlocked noautoreload", "Key 'Options' has unexpected value in label P4::Spec")
	assert.Equal(s.T(), label["Revision"], "@1234", "Key 'Revision' has unexpected value in label P4::Spec")
	assert.Equal(s.T(), label["View"], []string{"//depot/... ## Inline Comment", "## New Comment line"}, "Key 'View' has unexpected value in label P4::Spec")

	// Protect
	protect, err := s.p4api.ParseSpec("protect", PROTECTSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse protect spec: %v", err)
	}
	assert.Equal(s.T(), protect["Protections"], []string{"write user * * //...", "super user test * //..."}, "Key 'Protections' has unexpected value in protect P4::Spec")

	// Stream
	stream, err := s.p4api.ParseSpec("stream", STREAMSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse stream spec: %v", err)
	}

	assert.Equal(s.T(), stream["Stream"], "//stream/main", "Key 'Stream' has unexpected value in stream P4::Spec")
	assert.Equal(s.T(), stream["Update"], "2011/01/01 00:00:00", "Key 'Update' has unexpected value in stream P4::Spec")
	assert.Equal(s.T(), stream["Access"], "2012/01/01 00:00:00", "Key 'Access' has unexpected value in stream P4::Spec")
	assert.Equal(s.T(), stream["Owner"], "test", "Key 'Owner' has unexpected value in stream P4::Spec")
	assert.Equal(s.T(), stream["Name"], "main", "Key 'Name' has unexpected value in stream P4::Spec")
	assert.Equal(s.T(), stream["Parent"], "none", "Key 'Parent' has unexpected value in stream P4::Spec")
	assert.Equal(s.T(), stream["Type"], "mainline", "Key 'Type' has unexpected value in stream P4::Spec")
	assert.NotEmpty(s.T(), stream["Description"], "Key 'Description' missing from stream P4::Spec")
	assert.Equal(s.T(), stream["Options"], "allsubmit unlocked toparent fromparent", "Key 'Options' has unexpected value in stream P4::Spec")
	assert.Equal(s.T(), stream["Paths"], []string{"share ..."}, "Key 'Paths' has unexpected value in stream P4::Spec")
	assert.Equal(s.T(), stream["Remapped"], []string{"test/...	test_remap/..."}, "Key 'Remapped' has unexpected value in stream P4::Spec")
	assert.Equal(s.T(), stream["Ignored"], []string{"excluded/..."}, "Key 'Ignored' has unexpected value in stream P4::Spec")
	assert.Equal(s.T(), stream["View"], []string{"//stream/main/... ...", "//stream/main/test/... test_remap/... ## Inline Comment", "-//stream/...excluded/... ...excluded/...", "## New Comment line"}, "Key 'View' has unexpected value in stream P4::Spec")

	// Triggers
	trigger, err := s.p4api.ParseSpec("triggers", TRIGGERSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse triggers spec: %v", err)
	}

	assert.Equal(s.T(), trigger["Triggers"], []string{"name form-out change \"/some/script\""}, "Key 'Triggers' has unexpected value in triggers P4::Spec")

	// Typemap
	typemap, err := s.p4api.ParseSpec("typemap", TYPEMAPSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse typemap spec: %v", err)
	}

	assert.Equal(s.T(), typemap["TypeMap"], []string{"+l //....docx"}, "Key 'TypeMap' has unexpected value in typemap P4::Spec")

	// User
	user, err := s.p4api.ParseSpec("user", USERSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse user spec: %v", err)
	}

	assert.Equal(s.T(), user["User"], "test", "Key 'User' has unexpected value in user P4::Spec")
	assert.Equal(s.T(), user["Type"], "standard", "Key 'Type' has unexpected value in user P4::Spec")
	assert.Equal(s.T(), user["Email"], "test@test", "Key 'Email' has unexpected value in user P4::Spec")
	assert.Equal(s.T(), user["Update"], "2011/01/01 00:00:00", "Key 'Update' has unexpected value in user P4::Spec")
	assert.Equal(s.T(), user["Access"], "2012/01/01 00:00:00", "Key 'Access' has unexpected value in user P4::Spec")
	assert.Equal(s.T(), user["FullName"], "test", "Key 'FullName' has unexpected value in user P4::Spec")
	assert.Equal(s.T(), user["JobView"], "status=open", "Key 'JobView' has unexpected value in user P4::Spec")
	assert.Equal(s.T(), user["Password"], "testpass", "Key 'Password' has unexpected value in user P4::Spec")
	assert.Equal(s.T(), user["Reviews"], []string{"//depot/path/..."}, "Key 'Reviews' has unexpected value in user P4::Spec")

	// Server
	server, err := s.p4api.ParseSpec("server", SERVERSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse server spec: %v", err)
	}

	assert.Equal(s.T(), server["ServerID"], "418777F0-F7A5-439B-BAFC-E790E7286188", "Key 'ServerID' has unexpected value in server P4::Spec")
	assert.Equal(s.T(), server["Type"], "server", "Key 'Type' has unexpected value in server P4::Spec")
	assert.Equal(s.T(), server["Name"], "Maser", "Key 'Name' has unexpected value in server P4::Spec")
	assert.Equal(s.T(), server["Address"], "1666", "Key 'Address' has unexpected value in server P4::Spec")
	assert.Equal(s.T(), server["Services"], "standard", "Key 'Services' has unexpected value in server P4::Spec")
	assert.NotEmpty(s.T(), server["Description"], "Key 'Description' missing from server P4::Spec")

	// LDAP
	ldap, err := s.p4api.ParseSpec("ldap", LDAPSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse ldap spec: %v", err)
	}

	assert.Equal(s.T(), ldap["Name"], "myLDAPconfig", "Key 'Name' has unexpected value in ldap P4::Spec")
	assert.Equal(s.T(), ldap["Host"], "openldap.example.com", "Key 'Host' has unexpected value in ldap P4::Spec")
	assert.Equal(s.T(), ldap["Port"], "389", "Key 'Port' has unexpected value in ldap P4::Spec")
	assert.Equal(s.T(), ldap["Encryption"], "tls", "Key 'Encryption' has unexpected value in ldap P4::Spec")
	assert.Equal(s.T(), ldap["BindMethod"], "search", "Key 'BindMethod' has unexpected value in ldap P4::Spec")
	assert.Equal(s.T(), ldap["Options"], "nodowncase nogetattrs norealminusername", "Key 'Options' has unexpected value in ldap P4::Spec")
	assert.Equal(s.T(), ldap["SimplePattern"], "someuserid", "Key 'SimplePattern' has unexpected value in ldap P4::Spec")
	assert.Equal(s.T(), ldap["SearchBaseDN"], "ou=employees,dc=example,dc=com", "Key 'SearchBaseDN' has unexpected value in ldap P4::Spec")
	assert.Equal(s.T(), ldap["SearchFilter"], "(cn=%user%)", "Key 'SearchFilter' has unexpected value in ldap P4::Spec")
	assert.Equal(s.T(), ldap["SearchScope"], "subtree", "Key 'SearchScope' has unexpected value in ldap P4::Spec")
	assert.Equal(s.T(), ldap["GroupSearchScope"], "subtree", "Key 'GroupSearchScope' has unexpected value in ldap P4::Spec")

	// Remote
	remote, err := s.p4api.ParseSpec("remote", REMOTESPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse remote spec: %v", err)
	}

	assert.Equal(s.T(), remote["RemoteID"], "test_remote", "Key 'RemoteID' has unexpected value in remote P4::Spec")
	assert.Equal(s.T(), remote["Address"], "localhost:1666", "Key 'Address' has unexpected value in remote P4::Spec")
	assert.NotEmpty(s.T(), remote["Description"], "Key 'Description' has unexpected value in remote P4::Spec")
	assert.Equal(s.T(), remote["DepotMap"], []string{"//... //remote/test_depot/..."}, "Key 'DepotMap' has unexpected value in remote P4::Spec")

	// Repo
	repo, err := s.p4api.ParseSpec("repo", REPOSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse repo spec: %v", err)
	}

	assert.Equal(s.T(), repo["Repo"], "//GD1/pizza", "Key 'Repo' has unexpected value in repo P4::Spec")
	assert.Equal(s.T(), repo["Owner"], "test", "Key 'Owner' has unexpected value in repo P4::Spec")
	assert.Equal(s.T(), repo["Created"], "2024/12/18 14:57:16", "Key 'Created' has unexpected value in repo P4::Spec")
	assert.NotEmpty(s.T(), repo["Description"], "Key 'Description' has unexpected value in repo P4::Spec")
	assert.Equal(s.T(), repo["Options"], "lfs", "Key 'Options' has unexpected value in repo P4::Spec")
	assert.Equal(s.T(), repo["GconnMirrorStatus"], "disabled", "Key 'GconnMirrorStatus' has unexpected value in repo P4::Spec")
	assert.Equal(s.T(), repo["GconnMirrorHideFetchUrl"], "false", "Key 'GconnMirrorHideFetchUrl' has unexpected value in repo P4::Spec")

	// HotFiles
	hotfiles, err := s.p4api.ParseSpec("hotfiles", HOTFILESSPEC)
	if err != nil {
		s.T().Fatalf("Failed to parse hotfiles spec: %v", err)
	}

	assert.Equal(s.T(), hotfiles["HotFiles"], []string{"//depot/project1/...", "//depot/project2/... text"}, "Key 'HotFiles' has unexpected value in hotfiles P4::Spec")

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestServer() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")
	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	res, err := s.p4api.ServerUnicode()
	assert.Nil(s.T(), err, "Failed to get server unicode")
	assert.False(s.T(), res, "Server should not be in unicode mode")
	res, err = s.p4api.ServerCaseSensitive()
	assert.Nil(s.T(), err, "Failed to get server case sensitivity")
	assert.False(s.T(), res, "Server should not be in case sensitive mode")

	level, _ := s.p4api.ServerLevel()
	assert.Greater(s.T(), level, 0, "Server level should be non-zero")

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestOutput() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	s.createClient()

	// Files to create
	files := []string{"foo", "bar", "baz"}

	// Loop through each file and create it
	for _, fn := range files {
		// Construct the filename path
		filepath := fmt.Sprintf("%s.txt", fn)

		// Create and write to the file
		err := os.WriteFile(filepath, []byte("Test\n"), 0644)
		if err != nil {
			s.T().Fatalf("Failed to create file %s: %v", filepath, err)
		}

		// Simulate adding the file to Perforce
		_, _ = s.p4api.Run("add", filepath)
	}

	_, err = s.p4api.RunSubmit("-d", "test")
	assert.ErrorIs(s.T(), nil, err, "Failed to submit test")

	// P4MessageSeverity value
	// P4MESSAGE_EMPTY = 0
	// P4MESSAGE_INFO = 1
	// P4MESSAGE_WARN = 2
	// P4MESSAGE_FAILED = 3
	// P4MESSAGE_FATAL = 4

	// Run a 'p4 sync' and ignore the output, then run it again
	// and we should get a 'file(s) up-to-date' message that we
	// want to trap.
	_, _ = s.p4api.Run("sync")
	res, _ := s.p4api.Run("sync")

	// Check warnings array contains the warning as a string
	// assert( !p4.warnings.empty?, "Didn't get a warning!")

	assert.Equal(s.T(), res[0].(P4Message).lines[0].fmt, "File(s) up-to-date.", "Didn't get expected warning")

	// Now check messages array contains the message object

	w := res[0].(P4Message).lines[0]
	assert.Equal(s.T(), w.severity, P4MessageSeverity(2), "Severity was not E_WARN")
	assert.Equal(s.T(), w.code, 554768772, "Got the wrong message ID")

	// Sync to none and then sync to head - check number of info, warning
	// and error messages
	_, _ = s.p4api.Run("sync", "//...#0")
	s.p4api.SetTagged(false)
	res, _ = s.p4api.Run("sync", "//depot...")

	w = res[0].(P4Message).lines[0]
	assert.Equal(s.T(), w.severity, P4MessageSeverity(1), "Severity was not E_INFO")
	assert.Equal(s.T(), 3, len(res), "Wrong number of info messages")

	// test getting an error's dictionary (hash)
	res, _ = s.p4api.Run("dirs", "//this/is/a/path/that/does/not/exist/*")
	errs := res[0].(P4Message)
	assert.Equal(s.T(), errs.severity, P4MessageSeverity(3), "Severity was not E_FAILED")
	assert.Equal(s.T(), 1, len(errs.lines), "Got the wrong number of failed message")
	w = errs.lines[0]

	require.True(s.T(), len(errs.lines) > 0, "Empty dictionary")
	assert.NotEmpty(s.T(), w.code, "No message code present")
	assert.NotEmpty(s.T(), w.fmt, "No message format present")

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()

}

func (s *PerforceTestSuite) TestTrack() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")
	_, err := s.p4api.SetTrack(true)
	assert.Nil(s.T(), err, "Failed to set performance tracking")
	assert.True(s.T(), s.p4api.Track(), "Failed to set performance tracking")
	_, err = s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	assert.Condition(s.T(), func() bool {
		defer func() {
			if r := recover(); r != nil {
				s.T().Logf("Recovered from panic: %v", r)
			}
		}()

		_, _ = s.p4api.SetTrack(false)
		return s.p4api.Track() != false

	}, "Performance tracking cannot be changed")

	assert.True(s.T(), s.p4api.Track(), "Performance tracking cannot be changed")

	// Assert that the track output is not empty
	trackOutput, _ := s.p4api.Run("info")
	assert.IsType(s.T(), P4Track(""), trackOutput[1])
	assert.NotEmpty(s.T(), trackOutput, "No performance tracking reported")

	found := false

	for _, o := range trackOutput {
		if str, ok := o.(P4Track); ok && len(str) >= 3 && str[:3] == "rpc" {
			found = true
			break
		}
	}

	assert.True(s.T(), found, "Failed to report expected performance tracking output")

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestShelve() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	// Create test client
	s.createClient()

	// Create the directory "test_files"
	err := os.Mkdir("test_files", 0755)
	if err != nil {
		s.T().Fatalf("Failed to create directory: %v", err)
	}

	// Files to create
	files := []string{"foo", "bar", "baz"}

	// Loop through each file and create it
	for _, fn := range files {
		// Construct the filename path
		filepath := fmt.Sprintf("test_files/%s.txt", fn)

		// Create and write to the file
		err := os.WriteFile(filepath, []byte("This is a test file\n"), 0644)
		if err != nil {
			s.T().Fatalf("Failed to create file %s: %v", filepath, err)
		}

		// Simulate adding the file to Perforce
		_, _ = s.p4api.Run("add", filepath)
	}

	// Check if the files were created and added
	for _, fn := range files {
		filepath := fmt.Sprintf("test_files/%s.txt", fn)
		_, err := os.Stat(filepath)
		assert.False(s.T(), os.IsNotExist(err), fmt.Sprintf("File %s should exist", filepath))
	}

	_, err = s.p4api.RunSubmit("-d", "test")
	assert.ErrorIs(s.T(), nil, err, "Failed to submit test")

	_, _ = s.p4api.Run("sync", "#0")
	_, _ = s.p4api.Run("sync")

	// Edit test files
	for _, name := range files {
		filepath := fmt.Sprintf("test_files/%s.txt", name)
		_, _ = s.p4api.Run("edit", filepath)
		file, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, 0644)
		assert.ErrorIs(s.T(), nil, err, fmt.Sprintf("Failed to open file for editing: %s", name))

		_, err = file.WriteString("Change for a shelf\n")
		assert.ErrorIs(s.T(), nil, err, fmt.Sprintf("Failed to write to file: %s", name))
		file.Close()
	}

	// Create a pending change
	changeSpec, err := s.p4api.RunFetch("change")
	assert.ErrorIs(s.T(), nil, err, "Failed to get change spec")
	changeSpec["Description"] = "My shelf"

	// Shelve the lot
	shelveSpecList, err := s.p4api.RunShelve(changeSpec)
	assert.ErrorIs(s.T(), nil, err, "Failed to get shelve spec")
	shelveSpec := shelveSpecList[0]
	changeNum := shelveSpec["change"]

	// Revert local files
	_, _ = s.p4api.Run("revert", "test_files/...")
	results, err := s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 0, "Shouldn't have any open files")

	// Unshelve it again
	_, _ = s.p4api.Run("unshelve", "-s", changeNum.(string), "-f")
	results, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 3, "None or not all files unshelved")

	// And delete the shelve
	_, _ = s.p4api.Run("shelve", "-d", changeNum.(string))
	_, _ = s.p4api.Run("revert", "test_files/...")

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestStream() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	depotspec, _ := s.p4api.RunFetch("depot", "Stream")

	// Check server_level to ensure that the server support streams.
	level, _ := s.p4api.ServerLevel()
	if level >= 30 && s.p4api.ApiLevel() >= 70 {

		assert.True(s.T(), s.p4api.Streams(), "Streams are not enabled")

		// Create a new streams depot and make sure that it's listed.
		depotspec["Type"] = "stream"
		_, err := s.p4api.RunSave("depot", depotspec)
		assert.NoError(s.T(), err, "Failed to create streams depot")
		depots, err := s.p4api.Run("depots")
		require.NoError(s.T(), err, "Failed to fetch depots")
		require.Len(s.T(), depots, 2, "Streams depot not created")

		// Disable streams
		assert.True(s.T(), s.p4api.ApiLevel() >= 70, "API level (%d) too low", s.p4api.ApiLevel())
		s.p4api.SetStreams(false)
		assert.False(s.T(), s.p4api.Streams(), "Failed to disable streams")

		// Note: as of 2016.1, streams depot is no reason to need the streams
		// tag to include streams depots.
		depots, err = s.p4api.Run("depots")
		require.NoError(s.T(), err, "Failed to fetch depots")
		require.Len(s.T(), depots, 2, "Streams depot not included in depots command")

		// Enable streams and set the api_level < 70
		s.p4api.SetStreams(true)
		assert.True(s.T(), s.p4api.Streams(), "Failed to enable streams")
		oldLevel := s.p4api.ApiLevel()
		s.p4api.SetApiLevel(69)

		assert.True(s.T(), s.p4api.ApiLevel() < 70, "API level (%d) too high", s.p4api.ApiLevel())
		depots, err = s.p4api.Run("depots")
		require.NoError(s.T(), err, "Failed to fetch depots")
		require.Len(s.T(), depots, 2, "Streams depot not included in depots command")

		// reset the level
		s.p4api.SetApiLevel(oldLevel)

		// Fetch a stream from the server, check that
		// an 'extraTag' field (such as 'firmerThanParent' exists, and save
		// the spec
		streamSpec, _ := s.p4api.RunFetch("stream", "//Stream/MAIN")
		streamSpec["Paths"] = []string{"share ... ## Inline comment", "## Newline comment"}
		// assert.Contains(s.T(), streamSpec, "firmerThanParent", "'extraTag' field missing from spec.")
		streamSpec["Type"] = "mainline"
		msg, _ := s.p4api.RunSave("stream", streamSpec)

		assert.Contains(s.T(), msg.String(), "saved", "Failed to create a stream")
		nstreamSpec, _ := s.p4api.RunFetch("stream", "//Stream/MAIN")
		// After save and fetch, Paths should be an array
		paths, ok := nstreamSpec["Paths"].([]string)
		require.True(s.T(), ok, "Paths should be a []string array")
		require.GreaterOrEqual(s.T(), len(paths), 1, "Should have at least 1 path entry")
		assert.Equal(s.T(), "share ... ## Inline comment", paths[0])
		assert.Equal(s.T(), "## Newline comment", paths[1])

	} else {
		fmt.Println("\tTest Skipped: Streams requires a 2011.1 or later Perforce Server and P4API.")
	}
	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestSpecIterator() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	s.createClient()

	label, _ := s.p4api.RunFetch("label", "label1")
	assert.IsTypef(s.T(), Dictionary{}, label, "Expected type %T, but got %T", Dictionary{}, label)
	_, err = s.p4api.RunSave("label", label)
	assert.NoError(s.T(), err, "Failed to save label")

	label, _ = s.p4api.RunFetch("label", "label2")
	assert.IsTypef(s.T(), Dictionary{}, label, "Expected type %T, but got %T", Dictionary{}, label)
	_, err = s.p4api.RunSave("label", label)
	assert.NoError(s.T(), err, "Failed to save label")

	labelsSpec, _ := s.p4api.SpecIterator("labels")
	require.Len(s.T(), labelsSpec, 2, "Unexpected number of labels")

	for _, l := range labelsSpec {
		assert.IsTypef(s.T(), Dictionary{}, l, "Expected type %T, but got %T", Dictionary{}, l)
		if _, exists := l["Label"]; !exists {
			s.T().Errorf("Key '%s' not found in dictionary", "Label")
		}
	}

	labels, _ := s.p4api.SpecIterator("labels", "-e", "label1", "-m1")

	for _, l := range labels {
		assert.IsTypef(s.T(), Dictionary{}, l, "Expected type %T, but got %T", Dictionary{}, l)
		if _, exists := l["Label"]; !exists {
			s.T().Errorf("Key '%s' not found in dictionary", "Label")
		}
	}
	assert.Equal(s.T(), "label1", labels[0]["Label"], "Unexpected label in filtered result")
	require.Len(s.T(), labels, 1, "Unexpected number of labels")

	// 1. Create a branch
	branchSpec, _ := s.p4api.RunFetch("branch", "test_branch")
	branchSpec["View"] = []string{"//depot/src/... //depot/target/... ## Inline comment", "## Newline comment"}
	_, err = s.p4api.RunSave("branch", branchSpec)
	require.NoError(s.T(), err, "Failed to save branch")

	fetchedBranch, _ := s.p4api.RunFetch("branch", "test_branch")
	views, ok := fetchedBranch["View"].([]string)
	require.True(s.T(), ok, "View should be a []string array")
	require.GreaterOrEqual(s.T(), len(views), 1, "Should have at least 1 view entry")
	assert.Equal(s.T(), "//depot/src/... //depot/target/... ## Inline comment", views[0])
	assert.Equal(s.T(), "## Newline comment", views[1])

	// 2. Create a changelist
	changelist, err := s.p4api.RunFetch("change")
	require.NoError(s.T(), err)
	changelist["Description"] = "Test changelist created."
	_, err = s.p4api.RunSave("change", changelist)
	require.NoError(s.T(), err, "Failed to save changelist")

	// 3. Create a job
	jobSpec, err := s.p4api.RunFetch("job")
	require.NoError(s.T(), err)
	jobSpec["Description"] = "Test job created."
	_, err = s.p4api.RunSave("job", jobSpec)
	require.NoError(s.T(), err, "Failed to save job")

	// 4. Create a user
	s.p4api.SetUser("test_user")
	user, err := s.p4api.RunFetch("user")
	require.NoError(s.T(), err)
	user["FullName"] = "Test User"
	user["Email"] = "test_user@example.com"
	_, err = s.p4api.RunSave("user", user)
	require.NoError(s.T(), err, "Failed to save user")

	_, err = s.p4api.RunLogin("test_user")
	require.NoError(s.T(), err, "Failed to login as test_user")

	// 5. Create a group
	groupSpec, _ := s.p4api.RunFetch("group", "TestGroup")
	groupSpec["Users"] = []string{"user1"}
	_, err = s.p4api.RunSave("group", groupSpec)
	require.NoError(s.T(), err, "Failed to save group")

	// 6. Create a depot
	depotSpec, _ := s.p4api.RunFetch("depot", "test_depot")
	depotSpec["Type"] = "local"
	depotSpec["Description"] = "Test depot created."
	depotSpec["Map"] = "test_depot/..."
	_, err = s.p4api.RunSave("depot", depotSpec)
	require.NoError(s.T(), err, "Failed to save depot")

	// 7. Create a server
	serverSpec, _ := s.p4api.RunFetch("server", "test_server")
	serverSpec["Type"] = "server"
	serverSpec["Name"] = "myserver"
	serverSpec["Description"] = "Test server created."
	_, err = s.p4api.RunSave("server", serverSpec)
	require.NoError(s.T(), err, "Failed to save server")

	// 8. Create LDAP configuration
	ldapSpec, _ := s.p4api.RunFetch("ldap", "test_ldap")
	ldapSpec["Host"] = "ldap.example.com"
	ldapSpec["Port"] = "389"
	ldapSpec["BindMethod"] = "search"
	ldapSpec["SearchFilter"] = "(uid=test_user)"
	ldapSpec["SearchBaseDN"] = "ou=employees,dc=example,dc=com"
	ldapSpec["Owners"] = "test_user"
	_, err = s.p4api.RunSave("ldap", ldapSpec)
	require.NoError(s.T(), err, "Failed to save ldap")

	// 9. Create a remote
	remoteSpec, _ := s.p4api.RunFetch("remote", "test_remote")
	remoteSpec["Address"] = "localhost:1666"
	remoteSpec["Description"] = "Test remote created."
	remoteSpec["DepotMap"] = "//... //remote/test_depot/..."
	_, err = s.p4api.RunSave("remote", remoteSpec)
	require.NoError(s.T(), err, "Failed to save remote")

	// 10. Create a protection
	protectSpec, _ := s.p4api.RunFetch("protect")
	protectSpec["Protections"] = "super user test_user * //..."
	_, err = s.p4api.RunSave("protect", protectSpec)
	require.NoError(s.T(), err, "Failed to save protection")

	// 11. Create a repo
	depotSpec, _ = s.p4api.RunFetch("depot", "-t", "graph", "GD1")
	_, err = s.p4api.RunSave("depot", depotSpec)
	require.NoError(s.T(), err, "Failed to save graph depot")
	repoSpec, _ := s.p4api.RunFetch("repo", "//GD1/pizza")
	_, err = s.p4api.RunSave("repo", repoSpec)
	require.NoError(s.T(), err, "Failed to save repo")

	// 12. Create a stream
	s.p4api.SetStreams(true)
	depotspec, _ := s.p4api.RunFetch("depot", "Stream")
	depotspec["Type"] = "stream"
	_, err = s.p4api.RunSave("depot", depotspec)
	require.NoError(s.T(), err, "Failed to save stream depot")
	streamSpec, _ := s.p4api.RunFetch("stream", "//Stream/MAIN")
	streamSpec["Paths"] = []string{"share ... ## Inline comment", "## Newline comment"}
	streamSpec["Type"] = "mainline"
	_, err = s.p4api.RunSave("stream", streamSpec)
	require.NoError(s.T(), err, "Failed to save stream")

	// Ensure that the iterator does not raise an exception for all known specs
	// (except Streams, which for some reason does generate a warning...)

	for _, e := range []string{
		"clients",
		"labels",
		"branches",
		"changes",
		"streams",
		"jobs",
		"users",
		"groups",
		"depots",
		"servers",
		"ldaps",
		"remotes",
		"repos"} {
		if res, err := s.p4api.SpecIterator(e); err != nil {
			s.T().Errorf("%v not a known spec for iteration", err)
		} else {
			assert.Greater(s.T(), len(res), 0, "Unexpected number of specs %s", e)
		}
	}

	for _, e := range []string{"test", "nonexistent_spec"} {
		if _, err := s.p4api.SpecIterator(e); err == nil {
			s.T().Errorf("%v is a known spec for iteration", err)
		}
	}

	// Set up the client
	clientSpec, _ := s.p4api.RunFetch("client")
	_, err = s.p4api.RunSave("client", clientSpec)
	require.NoError(s.T(), err, "Failed to save client")

	// Add a test file
	filepath := "testfile.txt"

	// Create and write to the file
	err = os.WriteFile(filepath, []byte("This is a test file\n"), 0644)
	if err != nil {
		s.T().Fatalf("Failed to create file %s: %v", filepath, err)
	}

	// Simulate adding the file to Perforce
	_, _ = s.p4api.Run("add", filepath)

	changeSpec, _ := s.p4api.RunFetch("change")
	changeSpec["Description"] = "Add some test files\n"

	_, err = s.p4api.RunSubmit(changeSpec)
	assert.NoError(s.T(), err, "Failed to submit test")

	_, _ = s.p4api.Run("attribute", "-f", "-n", "test_tag_4", "-v", "set", "//depot/testfile.txt")

	// Confirm that attribute with number at the end gets converted to array
	// file := s.p4api.Run("fstat", "-Oa", "//depot/...")
	// fmt.Println("file", file)

	// res := file[0].(Dictionary)

	// if reflect.TypeOf(res["attr-test_tag_"]).Kind() != reflect.Slice {
	// 	s.T().Errorf("Expected file[0][\"attr-test_tag_\"] to be of type Array, but got %v", reflect.TypeOf(res["attr-test_tag_"]).Kind())
	// }

	// assert_kind_of(Array, file[0]["attr-test_tag_"])

	// Disable array conversion
	// p4.set_array_conversion = false

	// file = p4.run("fstat", "-Oa", "//depot/...")
	// assert_kind_of(String , file[0]["attr-test_tag_4"])

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestUnload() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	old_client := s.p4api.Client()
	level, _ := s.p4api.ServerLevel()
	if level >= 33 {
		// Create our test workspace
		s.createClient()

		// Create an unload depot
		depotSpec, err := s.p4api.RunFetch("depot", "unload_depot")

		if err != nil {
			fmt.Println("Failed to fetch depot spec: ", err)
			if pErr, ok := err.(*P4Message); ok {
				fmt.Println("Failed to fetch depot spec:--", pErr)
			}
		}

		depotSpec["Type"] = "unload"
		_, err = s.p4api.RunSave("depot", depotSpec)
		assert.NoError(s.T(), err, "Failed to create unload depot")

		// Ensure that the client is created
		clients, _ := s.p4api.Run("clients", "-e", s.p4api.Client())
		require.Len(s.T(), clients, 1, "Unexpected number of client workspaces.")

		// Add some files to the depot so we have something to work with
		err = os.Mkdir("test_files", 0755)
		if err != nil {
			s.T().Fatalf("Failed to create directory: %v", err)
		}

		// Files to create
		files := []string{"foo", "bar", "baz"}

		// Loop through each file and create it
		for _, fn := range files {
			// Construct the filename path
			filepath := fmt.Sprintf("test_files/%s.txt", fn)

			// Create and write to the file
			err := os.WriteFile(filepath, []byte("This is a test file\n"), 0644)
			if err != nil {
				s.T().Fatalf("Failed to create file %s: %v", filepath, err)
			}

			// Simulate adding the file to Perforce
			_, _ = s.p4api.Run("add", filepath)
		}

		_, err = s.p4api.RunSubmit("-d", "test")
		assert.ErrorIs(s.T(), nil, err, "Failed to submit test")

		sync, _ := s.p4api.Run("have")
		assert.Greater(s.T(), len(sync), 0, "Have list was empty!")

		// Unload the client workspace and check it was successful\
		_, _ = s.p4api.Run("unload", "-c", s.p4api.Client())
		clients, _ = s.p4api.Run("clients", "-U", "-e", s.p4api.Client())
		assert.Equal(s.T(), 1, len(clients), "Unload depot does not contain unloaded workspace.")

		// Reload the client workspace
		_, _ = s.p4api.Run("reload", "-c", s.p4api.Client())
		clients, _ = s.p4api.Run("clients", "-U", "-e", s.p4api.Client())
		assert.Equal(s.T(), 0, len(clients), "Unload depot does not contain unloaded workspace.")

		have, _ := s.p4api.Run("have")
		assert.Equal(s.T(), len(have), len(sync), "Unexpected number of files sync'd to reloaded workspace.")

	} else {
		fmt.Println("\tTest Skipped: Unload depot requires a 2012.2 or later Perforce Server.")
	}

	s.p4api.SetClient(old_client)

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestIgnore() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	// Create .p4ignore file and test.txt file, and add test.txt to .p4ignore
	err := os.WriteFile(".p4ignore", []byte("test.txt\n"), 0644)
	assert.NoError(s.T(), err, "Failed to create .p4ignore file")

	err = os.WriteFile("test.txt", []byte("This is a test file to be ignored by Perforce.\n"), 0644)
	assert.NoError(s.T(), err, "Failed to create test.txt file")

	s.p4api.SetIgnoreFile(".p4ignore")
	assert.Equal(s.T(), ".p4ignore", s.p4api.IgnoreFile(), "Unexpected value for ignore file")
	assert.False(s.T(), s.p4api.Ignored("foo"))
	assert.True(s.T(), s.p4api.Ignored("test.txt"), "test.txt should be ignored according to .p4ignore")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestGraphDepot() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	MIN_SERVER_LEVEL := 42
	MIN_API_LEVEL := 81

	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	depotSpec, _ := s.p4api.RunFetch("depot", "GD0")
	level, _ := s.p4api.ServerLevel()
	if level >= MIN_SERVER_LEVEL && s.p4api.ApiLevel() >= MIN_API_LEVEL {
		assert.True(s.T(), s.p4api.Graph(), "Failed to connect to Perforce server")

		// Create a new graph depot and make sure that it's listed.
		existing, _ := s.p4api.Run("depots")
		depotSpec["Type"] = "graph"
		_, err = s.p4api.RunSave("depot", depotSpec)
		assert.NoError(s.T(), err, "Failed to create graph depot")
		depots, _ := s.p4api.Run("depots")
		assert.Equal(s.T(), len(existing)+1, len(depots), "Graph depot not created")

		// Disable graph
		assert.True(s.T(), s.p4api.ApiLevel() >= MIN_API_LEVEL, "API level (%d) too low", s.p4api.ApiLevel())
		s.p4api.SetGraph(false)
		assert.False(s.T(), s.p4api.Graph(), "Failed to disable graph depots")
		depots, _ = s.p4api.Run("depots")
		assert.Equal(s.T(), len(depots), 1, "Graph depot included in depots command")

		// Enable graph and set the api_level < 81
		s.p4api.SetGraph(true)
		assert.True(s.T(), s.p4api.Graph(), "Failed to enable graph depots")
		oldLevel := s.p4api.ApiLevel()
		s.p4api.SetApiLevel(MIN_API_LEVEL - 1)
		assert.True(s.T(), s.p4api.ApiLevel() < MIN_API_LEVEL, "API level (%d) too high", s.p4api.ApiLevel())
		depots, _ = s.p4api.Run("depots")
		assert.Equal(s.T(), len(depots), 1, "Graph depot included in depots command")
		s.p4api.SetApiLevel(oldLevel)

	}
	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")

	_, err = s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")
	level, _ = s.p4api.ServerLevel()
	if level >= MIN_SERVER_LEVEL && s.p4api.ApiLevel() >= MIN_API_LEVEL {
		repos, _ := s.p4api.Run("repos")
		assert.Equal(s.T(), len(repos), 0, "Failed to get repos list")
	}
	ret, err = s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")

	// Repos tests
	_, err = s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")
	level, _ = s.p4api.ServerLevel()
	if level < MIN_SERVER_LEVEL || s.p4api.ApiLevel() < MIN_API_LEVEL {
		fmt.Println("Skipping test, server or api level is too low")
	} else {
		depotSpec, _ = s.p4api.RunFetch("depot", "-t", "graph", "GD1")
		_, err = s.p4api.RunSave("depot", depotSpec)
		assert.NoError(s.T(), err, "Failed to save graph depot")

		// create some users: u_dev, u_useless
		userSpec, _ := s.p4api.RunFetch("user", "u_dev")
		_, err = s.p4api.RunSave("user", userSpec, "-f")
		assert.NoError(s.T(), err, "Failed to save user")

		userSpec, _ = s.p4api.RunFetch("user", "u_useless")
		_, err = s.p4api.RunSave("user", userSpec, "-f")
		assert.NoError(s.T(), err, "Failed to save user")

		// set GD1 permissions: read acces for u_useless, create-repo for u_dev
		_, _ = s.p4api.Run("grant-permission", "-d", "GD1", "-p", "read", "-u", "u_useless")
		_, _ = s.p4api.Run("grant-permission", "-d", "GD1", "-p", "write-all", "-u", "u_dev")
		_, _ = s.p4api.Run("grant-permission", "-d", "GD1", "-p", "create-repo", "-u", "u_dev")

		perms, _ := s.p4api.Run("show-permission", "-d", "GD1")
		// just make sure there are 4, the original owner permission and the 3 we just set
		assert.Equal(s.T(), 4, len(perms), "Wrong number of permissions: %d", perms)

		// create a repo, leave permissions the defaults
		repoSpec, _ := s.p4api.RunFetch("repo", "//GD1/pizza")
		_, err = s.p4api.RunSave("repo", repoSpec)
		assert.NoError(s.T(), err, "Failed to save repo")
		repos, _ := s.p4api.Run("repos")
		assert.Equal(s.T(), 1, len(repos), "Wrong number of repos: : %d", len(repos))

		// check default permissions (5, original 4 from depot plus owner of repo)
		perms, _ = s.p4api.Run("show-permission", "-n", "//GD1/pizza")
		assert.Equal(s.T(), 5, len(perms), "Wrong number of permissions: %d", len(perms))

		// add one
		_, _ = s.p4api.Run("grant-permission", "-n", "//GD1/pizza", "-p", "write-ref", "-u", "u_useless")
		perms, _ = s.p4api.Run("show-permission", "-n", "//GD1/pizza")
		assert.Equal(s.T(), 6, len(perms), "Wrong number of permissions: %d", len(perms))

		// revoke it
		_, _ = s.p4api.Run("revoke-permission", "-n", "//GD1/pizza", "-p", "write-ref", "-u", "u_useless")
		perms, _ = s.p4api.Run("show-permission", "-n", "//GD1/pizza")
		assert.Equal(s.T(), 5, len(perms), "Wrong number of permissions: %d", len(perms))

		// check various users
		// return value is "none" or the permission we asked about
		perms, _ = s.p4api.Run("check-permission", "-n", "//GD1/pizza", "-p", "write-ref", "-u", "u_dev")
		assert.NotEqual(s.T(), perms[0].(Dictionary)["perm"], "none", "Wrong check-permission result")
		perms, _ = s.p4api.Run("check-permission", "-n", "//GD1/pizza", "-p", "write-ref", "-u", "u_useless")
		assert.Equal(s.T(), perms[0].(Dictionary)["perm0"], "none", "Wrong check-permission result")

		// make a branch (reference) restriction, only the current user can submit
		me := s.p4api.User()
		_, _ = s.p4api.Run("grant-permission", "-n", "//GD1/pizza", "-u", me, "-p", "restricted-ref", "-r",
			"refs/heads/master")

		// now check if u_dev can write-ref to that specific branch
		perms, _ = s.p4api.Run("check-permission", "-n", "//GD1/pizza", "-r", "refs/heads/master", "-p", "write-ref", "-u", "u_dev")
		assert.Equal(s.T(), perms[0].(Dictionary)["perm0"], "none", "Wrong check-permission result")

		// also check that u_dev can write to another random ref
		perms, _ = s.p4api.Run("check-permission", "-n", "//GD1/pizza", "-r", "refs/heads/delicious", "-p", "write-ref", "-u", "u_dev")
		assert.NotEqual(s.T(), perms[0].(Dictionary)["perm0"], "none", "Wrong check-permission result")
	}

	ret, err = s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestEvilTwin() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")
	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	// Prep workspace
	clientSpec, _ := s.p4api.RunFetch("client")
	clientSpec["Root"] = s.clientRoot
	clientSpec["Description"] = "Test client\n"
	_, err = s.p4api.RunSave("client", clientSpec)
	assert.NoError(s.T(), err, "Failed to save client")
	err = os.Mkdir("A", 0755)
	if err != nil {
		s.T().Fatalf("Failed to create directory: %v", err)
	}

	// Adding
	filepath := "A/fileA"
	err = os.WriteFile(filepath, []byte("Original content\n"), 0644)
	if err != nil {
		s.T().Fatalf("Failed to create file %s: %v", filepath, err)
	}

	_, _ = s.p4api.Run("add", filepath)
	_, err = s.p4api.RunSubmit("-d", "adding fileA")
	assert.NoError(s.T(), err, "Failed to submit")

	// Branching
	branchSpec, _ := s.p4api.RunFetch("branch", "-o", "evil-twin-test")
	branchSpec["View"] = []string{"//depot/A/... //depot/B/... ## Inline comment", "## Newline comment"}
	_, err = s.p4api.RunSave("branch", branchSpec)
	assert.NoError(s.T(), err, "Failed to save branch")
	fetchedBranch, _ := s.p4api.RunFetch("branch", "evil-twin-test")
	views, ok := fetchedBranch["View"].([]string)
	require.True(s.T(), ok, "View should be a []string array")
	require.GreaterOrEqual(s.T(), len(views), 1, "Should have at least 1 view entry")
	assert.Equal(s.T(), "//depot/A/... //depot/B/... ## Inline comment", views[0])
	assert.Equal(s.T(), "## Newline comment", views[1])

	_, _ = s.p4api.Run("integ", "-b", "evil-twin-test")
	_, err = s.p4api.RunSubmit("-d", "integrating")
	assert.NoError(s.T(), err, "Failed to submit")

	// Moving
	_, _ = s.p4api.Run("edit", "A/fileA")
	_, _ = s.p4api.Run("move", "-f", "A/fileA", "A/fileA1")
	_, err = s.p4api.RunSubmit("-d", "moving")
	assert.NoError(s.T(), err, "Failed to submit")

	// Re-adding origianl
	filepath = "A/fileA"
	err = os.WriteFile(filepath, []byte("Re-added A\n"), 0644)
	if err != nil {
		s.T().Fatalf("Failed to create file %s: %v", filepath, err)
	}

	_, _ = s.p4api.Run("add", filepath)
	_, err = s.p4api.RunSubmit("-d", "re-adding")
	assert.NoError(s.T(), err, "Failed to submit")

	// Second merge
	_, _ = s.p4api.Run("merge", "-b", "evil-twin-test")
	_, err = s.p4api.RunSubmit("-d", "integrating")
	assert.Contains(s.T(), err.Error(), "Submit failed", "Unexpected number of open files")

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()

}

func (s *PerforceTestSuite) TestUnicode() {
	s.enableUnicode()
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	// s.p4api.SetEnv("P4CHARSET", "utf-8")

	// return
	// res, err := s.p4api.Connect()
	// res, err = s.p4api.Connect()

	res, err := s.p4api.SetCharset("iso8859-1")
	assert.NoError(s.T(), err, "Failed to set charset")
	assert.True(s.T(), res, "Failed to set charset")

	_, err = s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	res, err = s.p4api.SetEnv("iso8859", "blas")
	assert.NoError(s.T(), err, "Failed to set charset")
	assert.True(s.T(), res, "Failed to set charset")

	// assert.True(s.T(), s.p4api.ServerUnicode(), "Server should be in unicode mode")
	s.createClient()

	// Add a test file with a £ sign in it. That has the high bit set
	// so we can test it in both iso8859-1 and utf-8

	// Add a test file
	filepath := "unicode.txt"

	// Create and write to the file
	err = os.WriteFile(filepath, []byte("This file cost \xa31\n"), 0644)
	assert.NoError(s.T(), err, "Failed to create file")
	_, _ = s.p4api.Run("add", filepath)
	results, err := s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 1, "There should be only 1 file open")
	_, _ = s.p4api.Run("submit", "-d", "Add unicode test file")
	results, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 0, "There should be no files open")

	// Now remove the file from the workspace, disconnect, switch to
	// utf8, reconnect and resync the file. Then we'll print it and
	// see that the content contains the unicode sequence for the £
	// symbol.

	_, _ = s.p4api.Run("sync", fmt.Sprintf("%s#none", filepath))

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")

	_, _ = s.p4api.SetCharset("utf8")
	_, err = s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")
	_, _ = s.p4api.Run("sync")

	// Read the file
	data, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Remove trailing newline characters
	buf := strings.TrimSpace(string(data))

	assert.Equal(s.T(), "This file cost £1", buf, "Unicode support broken")

	ret, err = s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

/*
############################################
  Start of the test for the handler command
############################################
*/

type NewOutputHandler struct {
	statOutput    []Dictionary
	infoOutput    []string
	messageOutput []P4Message
}

func (h *NewOutputHandler) HandleStat(dict Dictionary) P4OutputHandlerResult {
	h.statOutput = append(h.statOutput, dict)
	return P4OUTPUTHANDLER_HANDLED
}

func (h *NewOutputHandler) HandleText(info string) P4OutputHandlerResult {
	h.infoOutput = append(h.infoOutput, info)
	return P4OUTPUTHANDLER_REPORT
}

func (h *NewOutputHandler) HandleMessage(msg P4Message) P4OutputHandlerResult {
	h.messageOutput = append(h.messageOutput, msg)
	return P4OUTPUTHANDLER_REPORT
}

func (h *NewOutputHandler) HandleTrack(data string) P4OutputHandlerResult {
	return P4OUTPUTHANDLER_REPORT
}

func (h *NewOutputHandler) HandleSpec(data Dictionary) P4OutputHandlerResult {
	return P4OUTPUTHANDLER_REPORT
}

func (h *NewOutputHandler) HandleBinary(data []byte) P4OutputHandlerResult {
	return P4OUTPUTHANDLER_REPORT
}

func (s *PerforceTestSuite) TestOutputHandler() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	s.createClient()

	// Create the directory "test_files"
	err = os.Mkdir("handler_files", 0755)
	if err != nil {
		s.T().Fatalf("Failed to create directory: %v", err)
	}

	// Files to create
	files := []string{"foo", "bar", "baz"}

	// Loop through each file and create it
	for _, fn := range files {
		// Construct the filename path
		filepath := fmt.Sprintf("handler_files/%s.txt", fn)

		// Create and write to the file
		err := os.WriteFile(filepath, []byte("This is a test file\n"), 0644)
		if err != nil {
			s.T().Fatalf("Failed to create file %s: %v", filepath, err)
		}
		_, _ = s.p4api.Run("add", filepath)
	}

	results, err := s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 3, "Unexpected number of open files")
	change_specs, err := s.p4api.RunFetch("change")

	if err != nil {
		assert.Fail(s.T(), "failed to get change dict")
	}

	assert.IsType(s.T(), Dictionary{}, change_specs, "Change form is not a dict")
	change_specs["Description"] = "Add some test files\n"
	_, err = s.p4api.RunSubmit(change_specs)
	assert.NoError(s.T(), err, "Failed to submit test")

	// Ensure no files are open
	results, err = s.p4api.Run("opened")
	require.NoError(s.T(), err, "Failed to run 'opened'")
	require.Len(s.T(), results, 0, "Shouldn't have any open files")

	// Create an instance of our new output handler
	handler := &NewOutputHandler{}
	s.p4api.SetHandler(handler)

	// Check that the output goes into the Handler object and that the
	// Handler object contains the correct number of files and messages.
	filesResult, _ := s.p4api.Run("files", "handler_files/...")
	assert.Equal(s.T(), 0, len(filesResult), "Does not return empty list")
	assert.Equal(s.T(), len(files), len(handler.statOutput), "Less files than expected")
	assert.Equal(s.T(), 0, len(handler.messageOutput), "Unexpected messages")

	s.p4api.SetHandler(nil)

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

/*
############################################
  Start of the test for the progress command
############################################
*/

type SubmitP4Progress struct {
	P4Progress
	types     []int
	descs     []string
	totals    []int64
	positions []int64
	fails     []bool
}

func (p *SubmitP4Progress) Init(progress_type int) {
	p.types = append(p.types, progress_type)
}

func (p *SubmitP4Progress) Description(desc string, units int) {
	p.descs = append(p.descs, desc)
}

func (p *SubmitP4Progress) Total(total int64) {
	p.totals = append(p.totals, total)
}

func (p *SubmitP4Progress) Update(position int64) {
	p.positions = append(p.positions, position)
}

func (p *SubmitP4Progress) Done(failed bool) {
	p.fails = append(p.fails, failed)
}

type SyncProgress struct {
	P4Progress
	types     int
	positions int64
	fails     bool
}

func (op *SyncProgress) Init(progress_type int) {
	op.types = progress_type
}

func (op *SyncProgress) Description(desc string, units int) {
	// Do nothing
}

func (op *SyncProgress) Total(total int64) {
	// Do nothing
}

func (op *SyncProgress) Update(position int64) {
	op.positions = position
}

func (op *SyncProgress) Done(failed bool) {
	op.fails = failed
}

type SyncOutput struct {
	P4OutputHandlerResult
	totalFiles []int
	totalSizes []int
}

func (op *SyncOutput) HandleBinary(data []byte) P4OutputHandlerResult {
	return P4OUTPUTHANDLER_REPORT
}

func (op *SyncOutput) HandleSpec(data Dictionary) P4OutputHandlerResult {
	return P4OUTPUTHANDLER_REPORT
}

func (op *SyncOutput) HandleStat(dict Dictionary) P4OutputHandlerResult {
	if totalFileCountStr, ok := dict["totalFileCount"].(string); ok {
		if totalFileCount, err := strconv.Atoi(totalFileCountStr); err == nil {
			op.totalFiles = append(op.totalFiles, totalFileCount)
		} else {
			op.totalFiles = append(op.totalFiles, 0)
		}
	} else {
		op.totalFiles = append(op.totalFiles, 0)
	}
	if sizeStr, ok := dict["totalFileSize"].(string); ok {
		if size, err := strconv.Atoi(sizeStr); err == nil {
			op.totalSizes = append(op.totalSizes, size)
		} else {
			op.totalSizes = append(op.totalSizes, 0)
		}
	} else {
		op.totalSizes = append(op.totalSizes, 0)
	}
	return P4OUTPUTHANDLER_HANDLED
}

func (op *SyncOutput) HandleText(info string) P4OutputHandlerResult {
	return P4OUTPUTHANDLER_REPORT
}

func (op *SyncOutput) HandleMessage(msg P4Message) P4OutputHandlerResult {
	return P4OUTPUTHANDLER_REPORT
}

func (op *SyncOutput) HandleTrack(data string) P4OutputHandlerResult {
	return P4OUTPUTHANDLER_REPORT
}

func (s *PerforceTestSuite) TestProgress() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")
	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	s.createClient()

	level, _ := s.p4api.ServerLevel()
	if level >= 33 {
		submitProg := &SubmitP4Progress{}
		s.p4api.SetProgress(submitProg)

		// Create the directory "test_files"
		err := os.Mkdir("progress_files", 0755)
		if err != nil {
			s.T().Fatalf("Failed to create directory: %v", err)
		}

		// Loop through each file and create it
		total := 50
		for n := range total {
			// Construct the filename path
			filepath := fmt.Sprintf("progress_files/file%d.txt", n)

			// Create and write to the file
			err := os.WriteFile(filepath, []byte(strings.Repeat("*", 1024)), 0644)
			if err != nil {
				s.T().Fatalf("Failed to create file %s: %v", filepath, err)
			}
			_, _ = s.p4api.Run("add", filepath)
		}

		openedFiles, err := s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to fetch opened files")
		require.Len(s.T(), openedFiles, 50, "Unexpected number of open files")
		change_specs, err := s.p4api.RunFetch("change")

		if err != nil {
			assert.Fail(s.T(), "failed to get change dict")
		}

		assert.IsType(s.T(), Dictionary{}, change_specs, "Change form is not a spec")
		change_specs["Description"] = "\nFiles: \n" // Set an empty description, which is usually required
		// Attempt to format the spec
		res, err := s.p4api.FormatSpec("change", change_specs)
		if err != nil {
			assert.Fail(s.T(), "Failed to format spec")
		}
		s.p4api.SetInput(res)
		_, err = s.p4api.Run("submit", "-i")
		assert.NoError(s.T(), err, "Failed to submit test")

		assert.Equal(s.T(), total, len(submitProg.types), "Did not receive %d progress init calls", total)
		assert.Equal(s.T(), total, len(submitProg.descs), "Did not receive %d progress description calls", total)
		assert.Equal(s.T(), total, len(submitProg.totals), "Did not receive %d progress total calls", total)
		assert.Equal(s.T(), total, len(submitProg.positions), "Did not receive %d progress update calls", total)
		assert.Equal(s.T(), total, len(submitProg.fails), "Did not receive %d progress done calls", total)

		// Ensure no files are open and that all files are present in the depot
		results, err := s.p4api.Run("opened")
		require.NoError(s.T(), err, "Failed to run 'opened'")
		require.Len(s.T(), results, 0, "Unexpected number of open files")
		results, err = s.p4api.Run("files", "progress_files/...")
		require.NoError(s.T(), err, "Failed to run 'files'")
		assert.Equal(s.T(), total, len(results), "Does not return empty list")

		// Quiet sync surpressed all info messages prior to 2014.1, so
		// this test will fail against 2012.2 - 2013.3 servers.  Now skip
		// those versions as the behaviour in the server has changed.
		level, _ := s.p4api.ServerLevel()
		if level >= 37 {
			progCallback := &SyncProgress{}
			s.p4api.SetProgress(progCallback)
			opCallback := &SyncOutput{}
			s.p4api.SetHandler(opCallback)

			_, _ = s.p4api.Run("sync", "-f", "-q", "//...")
			assert.Equal(s.T(), int(progCallback.positions), opCallback.totalFiles[0], "Total does not match position %d <> %d", opCallback.totalFiles, progCallback.positions)
			assert.Equal(s.T(), total, int(progCallback.positions), "Total does not match position %d <> %d", total, progCallback.positions)
		}

		s.p4api.SetProgress(nil)

	}
	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

/*
############################################
  Start of the test for the trust command
############################################
*/

func (s *PerforceTestSuite) TestTrust() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	port, teardown := s.setupTrust()
	defer teardown()

	assert.Equal(s.T(), s.p4api.Port(), port, "Should already be set correctly")

	s.testTrustFile()
	s.testDefaultTrust()
	s.p4api.Close()
}

func (s *PerforceTestSuite) setupTrust() (string, func()) {
	// create the ssl dir, needs to be "secure"
	sslDir := s.serverRoot + "/ssl"
	err := os.MkdirAll(sslDir, 0700)
	require.NoError(s.T(), err, "Failed to create SSL directory")

	// launch a p4d process to create a fingerprint file
	cmd := exec.Command("p4d", "-Gc")
	cmd.Env = append(os.Environ(), "P4SSLDIR="+sslDir)
	err = cmd.Run()
	require.NoError(s.T(), err, "Failed to create fingerprints")

	// pick a random port from 1024 -> 65000
	var randPort string
	retries := 10
	for retries > 0 {
		randPort = fmt.Sprintf("ssl:localhost:%d", 1024+rand.Intn(65000-1024))
		// launch a p4d to run trust against
		cmd = exec.Command("p4d", "-p", randPort)
		cmd.Env = append(os.Environ(), "P4SSLDIR="+sslDir)
		err = cmd.Start()
		if err == nil {
			// sleep for a bit (hack)
			time.Sleep(1 * time.Second)
			// nil indicates that the pid is still running
			if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
				break
			}
		}
		retries--
	}

	require.NotZero(s.T(), retries, "Could not find available port, test will fail")

	// also override the p4 to the ssl port-based p4d
	s.p4api.Close()
	s.p4api = New()
	s.p4api.SetPort(randPort)

	// dont connect, accepting the trust is part of the test

	teardown := func() {
		// send the shutdown signal
		// "Stopping #{@pid}"
		if cmd.Process != nil {
			err = cmd.Process.Kill()
			require.NoError(s.T(), err, "Failed to kill p4d")

			// wait for the process to exit
			// Note: After Kill(), Wait() may return a non-nil error on some platforms
			// (e.g., "signal: killed" on Unix, or an exit status on Windows). This is
			// expected and should not fail the test. We therefore ignore any error here.
			_ = cmd.Wait()
		}
		os.RemoveAll(s.serverRoot)
	}

	return randPort, teardown
}

func (s *PerforceTestSuite) testTrustFile() {
	// set a custom trust file
	trustFile := ".p4trust-test"
	s.p4api.SetTrustFile(trustFile)

	_, err := os.Stat(trustFile)
	assert.True(s.T(), os.IsNotExist(err), "Trust file should not exist")

	retries := 60
	ret, _ := s.p4api.Connect()
	for retries > 0 && !ret {
		time.Sleep(1 * time.Second)
		retries--
		ret, _ = s.p4api.Connect()
	}

	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	o, _ := s.p4api.Run("trust", "-y")
	// verify that our file has the fingerprint?
	// see the file is not empty
	assert.Contains(s.T(), o[1].(P4Data), "Added trust for P4PORT")
	fileInfo, err := os.Stat(trustFile)
	assert.NoError(s.T(), err, "Failed to stat trust file")
	assert.True(s.T(), fileInfo.Size() > 0, "Trust file size")

	// final is to just run a command and see that it succeeds
	info, _ := s.p4api.Run("info")
	assert.NotEmpty(s.T(), info, "Info command returned empty result")

	ret, err = s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
}

func (s *PerforceTestSuite) testDefaultTrust() {
	// verify that we get a "you need to accept" when connecting
	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	info, _ := s.p4api.Run("info")
	assert.NotEmpty(s.T(), info, "Info command returned empty result")

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")

	// run trust
	_, err = s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	o, _ := s.p4api.Run("trust", "-y")
	assert.Contains(s.T(), o[0].(P4Data), "Trust already established")

	info, _ = s.p4api.Run("info")
	assert.NotEmpty(s.T(), info, "Info command returned empty result")

	ret, err = s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
}

/*
############################################
  Start of the test for the sso command
############################################
*/

func (s *PerforceTestSuite) SSOSetup() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	// verify that we get a "you need to accept" when connecting
	triggersSpec, _ := s.p4api.RunFetch("triggers")
	assert.Empty(s.T(), triggersSpec, "Triggers should be empty")
	triggersSpec["Triggers"] = []string{"loginsso auth-check-sso auth pass"}
	msg, _ := s.p4api.RunSave("triggers", triggersSpec)
	assert.Equal(s.T(), "Triggers saved.", msg.String(), "Failed to save triggers")
	fetchedTriggersSpec, _ := s.p4api.RunFetch("triggers")
	// Verify the trigger was saved correctly
	triggers, ok := fetchedTriggersSpec["Triggers"].([]string)
	require.True(s.T(), ok, "Triggers should be a []string array")
	require.Len(s.T(), triggers, 1, "Should have 1 trigger")
	assert.Equal(s.T(), "loginsso auth-check-sso auth pass", triggers[0])

	// Set the log so we dont flood stderr
	res, _ := s.p4api.Run("configure", "set", "P4LOG=log")
	assert.Equal(s.T(), 1, len(res), "Failed to set log")
	assert.Equal(s.T(), Dictionary{"Action": "set", "Name": "P4LOG", "ServerName": "any", "Value": "log"}, res[0])

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) SSOTestDefault() {
	assert.NotNil(s.T(), s.p4api, "Failed to create Perforce client")

	_, err := s.p4api.Connect()
	assert.Nil(s.T(), err, "Failed to connect to Perforce server")
	assert.True(s.T(), s.p4api.Connected(), "Failed to connect to Perforce server")

	// assert_equal( nil, p4.loginsso )
	assert.Equal(s.T(), Dictionary{}, s.p4api, "SSO vars should be empty")
	//     begin
	//       p4.run_login( 'Passw0rd' )
	//     rescue P4Exception => e
	//       assert( e.to_s.include?("[Error]: Perforce password (P4PASSWD) invalid or unset."), "Exception thrown: #{e}" )
	//     end
	//     assert_equal( {}, p4.ssovars )

	ret, err := s.p4api.Disconnect()
	assert.True(s.T(), ret, "should disconnect")
	assert.Nil(s.T(), err, "should disconnect")
	s.p4api.Close()
}

func (s *PerforceTestSuite) TestSSO() {
	s.SSOSetup()
}

func TestPerforceTestSuite(t *testing.T) {
	suite.Run(t, new(PerforceTestSuite))
}
