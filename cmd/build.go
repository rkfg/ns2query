package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	project = "ns2query"
	tmp     = "tmp"
	release = "release"
)

type buildConfig struct {
	os       string
	arch     string
	suffix   string
	osSuffix string
}

func (bc buildConfig) binaryName() string {
	return project + "." + bc.suffix + bc.osSuffix
}

func (bc buildConfig) zipName() string {
	return project + "." + bc.suffix + ".zip"
}

var configs = []buildConfig{
	{os: "linux", arch: "amd64", suffix: "linux.amd64"},
	{os: "linux", arch: "386", suffix: "linux.386"},
	{os: "linux", arch: "mipsle", suffix: "linux.mipsel"},
	{os: "linux", arch: "arm", suffix: "linux.arm"},
	{os: "windows", arch: "amd64", suffix: "win64", osSuffix: ".exe"},
	{os: "windows", arch: "386", suffix: "win32", osSuffix: ".exe"},
}

func pack(cfg buildConfig, wg *sync.WaitGroup) {
	zipFile := path.Join("release", cfg.zipName())
	f, err := os.Create(zipFile)
	if err != nil {
		panic(fmt.Sprintf("Build failed! Error creating archive '%s': %s", zipFile, err))
	}
	defer f.Close()
	zipWriter := zip.NewWriter(f)
	defer zipWriter.Close()
	binary, err := os.Open(cfg.binaryName())
	if err != nil {
		panic(fmt.Sprintf("Build failed! Can't open binary %s: %s", cfg.binaryName(), err))
	}
	fi, err := binary.Stat()
	if err != nil {
		panic(fmt.Sprintf("Build failed! Binary %s is inaccessible: %s", cfg.binaryName(), err))
	}
	fih, err := zip.FileInfoHeader(fi)
	if err != nil {
		panic(fmt.Sprintf("Build failed! Error getting file info for zip %s: %s", cfg.zipName(), err))
	}
	fih.Method = zip.Deflate
	writer, err := zipWriter.CreateHeader(fih)
	if err != nil {
		panic(fmt.Sprintf("Build failed! Error creating a writer for %s: %s", cfg.zipName(), err))
	}
	_, err = io.Copy(writer, binary)
	if err != nil {
		panic(fmt.Sprintf("Build failed! Error compressing %s to %s: %s", cfg.binaryName(), cfg.zipName(), err))
	}
	os.Remove(cfg.binaryName())
	wg.Done()
}

func build(cfg buildConfig, versionFlags string, wg *sync.WaitGroup) {
	cmd := exec.Command("go", "build", "-ldflags", "-s -w "+versionFlags+" -extldflags -static", "-o", cfg.binaryName())
	cmd.Env = append(os.Environ(), "GOOS="+cfg.os, "GOARCH="+cfg.arch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("Build failed! Config: %+v, error: %s, args: %s, output:\n%s", cfg, err, cmd.Args, out))
	}
	pack(cfg, wg)
	wg.Done()
}

func versionFlags() string {
	date := time.Now().Format("01-02-2006 15:04:05 -0700 MST")
	result := fmt.Sprintf(`-X "main.date=%s"`, date)
	version := ""
	tag, err := exec.Command("git", "tag", "--contains", "HEAD").Output()
	if err != nil || string(tag) == "" {
		branch, err := exec.Command("git", "branch", "--show-current").Output()
		if err != nil || string(branch) == "" {
			commit, err := exec.Command("git", "log", "--pretty=format:%h", "-n1").Output()
			if err == nil && string(commit) != "" {
				version = fmt.Sprintf("commit %s", commit)
			}
		} else {
			version = fmt.Sprintf("branch %s", branch)
		}
	} else {
		version = string(tag)
	}
	if version != "" {
		result = fmt.Sprintf(`%s -X "main.version=%s"`, result, strings.TrimSuffix(version, "\n"))
	}
	return result
}

func main() {
	os.RemoveAll(release)
	os.MkdirAll(release, 0755)
	wg := sync.WaitGroup{}
	flags := versionFlags()
	for _, cfg := range configs {
		wg.Add(2)
		go build(cfg, flags, &wg)
	}
	wg.Wait()
}
