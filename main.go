package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-ini/ini"
)

type Release struct {
	Assets []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func main() {
	// Read password from file
	password, err := readPasswordFromFile("password.txt")
	if err != nil {
		fmt.Println("Error reading password file:", err)
		return
	}

	// GitHub API URL for the latest release
	apiURL := "https://api.github.com/repos/LukeYui/EldenRingSeamlessCoopRelease/releases/latest"

	// Get the latest release information
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("Error fetching release info:", err)
		return
	}
	defer resp.Body.Close()

	var release Release
	err = json.NewDecoder(resp.Body).Decode(&release)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	if len(release.Assets) == 0 {
		fmt.Println("No assets found in the latest release")
		return
	}

	// Download the zip file
	zipURL := release.Assets[0].BrowserDownloadURL
	zipResp, err := http.Get(zipURL)
	if err != nil {
		fmt.Println("Error downloading zip file:", err)
		return
	}
	defer zipResp.Body.Close()

	zipFile, err := os.CreateTemp("", "release-*.zip")
	if err != nil {
		fmt.Println("Error creating temp file:", err)
		return
	}
	defer os.Remove(zipFile.Name())

	_, err = io.Copy(zipFile, zipResp.Body)
	if err != nil {
		fmt.Println("Error saving zip file:", err)
		return
	}
	zipFile.Close()

	// Unzip the file to the current directory
	err = unzip(zipFile.Name(), "./")
	if err != nil {
		fmt.Println("Error unzipping file:", err)
		return
	}

	// Modify the ersc_settings.ini file
	iniPath := "SeamlessCoop/ersc_settings.ini"
	cfg, err := ini.Load(iniPath)
	if err != nil {
		fmt.Println("Error loading INI file:", err)
		return
	}

	// Update the existing cooppassword setting in the [PASSWORD] section
	section, err := cfg.GetSection("PASSWORD")
	if err != nil {
		// If the section doesn't exist, create it
		section, err = cfg.NewSection("PASSWORD")
		if err != nil {
			fmt.Println("Error creating PASSWORD section:", err)
			return
		}
	}

	key, err := section.GetKey("cooppassword")
	if err != nil {
		// If the key doesn't exist, create it
		_, err = section.NewKey("cooppassword", password)
		if err != nil {
			fmt.Println("Error creating cooppassword key:", err)
			return
		}
	} else {
		// If the key exists, update its value
		key.SetValue(password)
	}

	// Save the changes
	err = cfg.SaveTo(iniPath)
	if err != nil {
		fmt.Println("Error saving INI file:", err)
		return
	}

	fmt.Println("Successfully downloaded, unzipped, and modified the mod files.")
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
    //if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
		//	return fmt.Errorf("illegal file path: %s", fpath)
		//}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func readPasswordFromFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}
