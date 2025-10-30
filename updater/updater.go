package updater

import (
	"context"
	"fmt"
	"runtime"
	"time"

	selfupdate "github.com/creativeprojects/go-selfupdate"
	"github.com/google/go-github/v57/github"
)

const (
	repoOwner = "lsherman98"
	repoName  = "ytrss-cli"
)

func CheckForUpdate(currentVersion string) (*selfupdate.Release, bool, error) {
	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug(fmt.Sprintf("%s/%s", repoOwner, repoName)))
	if err != nil {
		return nil, false, fmt.Errorf("error checking for updates: %w", err)
	}

	if !found {
		return nil, false, fmt.Errorf("no releases found")
	}

	if latest.LessOrEqual(currentVersion) {
		return latest, false, nil
	}

	return latest, true, nil
}

func DoSelfUpdate(currentVersion string) error {
	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug(fmt.Sprintf("%s/%s", repoOwner, repoName)))
	if err != nil {
		return fmt.Errorf("error checking for updates: %w", err)
	}

	if !found {
		return fmt.Errorf("no releases found")
	}

	if latest.LessOrEqual(currentVersion) {
		fmt.Printf("Current version %s is already the latest\n", currentVersion)
		return nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("could not locate executable path: %w", err)
	}

	if err := selfupdate.UpdateTo(context.Background(), latest.AssetURL, latest.AssetName, exe); err != nil {
		return fmt.Errorf("error updating binary: %w", err)
	}

	fmt.Printf("Successfully updated to version %s\n", latest.Version())
	return nil
}

func GetLatestReleaseInfo() (string, string, error) {
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), repoOwner, repoName)
	if err != nil {
		return "", "", err
	}

	version := release.GetTagName()
	notes := release.GetBody()

	return version, notes, nil
}

func ShouldCheckForUpdate(lastCheck time.Time) bool {
	return time.Since(lastCheck) > 24*time.Hour
}

func GetAssetName(version string) string {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	ext := ".tar.gz"
	if osName == "windows" {
		ext = ".zip"
	}

	return fmt.Sprintf("ytrss-cli_%s_%s_%s%s", version, osName, arch, ext)
}

func CheckAndUpdate(currentVersion string) (bool, error) {
	if currentVersion == "dev" {
		return false, nil
	}

	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug(fmt.Sprintf("%s/%s", repoOwner, repoName)))
	if err != nil {
		return false, nil
	}

	if !found {
		return false, nil
	}

	if latest.LessOrEqual(currentVersion) {
		return false, nil
	}

	fmt.Printf("🎉 New version available: %s (current: %s)\n", latest.Version(), currentVersion)
	fmt.Println("Updating...")

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return false, fmt.Errorf("could not locate executable path: %w", err)
	}

	if err := selfupdate.UpdateTo(context.Background(), latest.AssetURL, latest.AssetName, exe); err != nil {
		return false, fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("✅ Successfully updated to version %s!\n", latest.Version())
	fmt.Println("Please restart the application to use the new version.")
	return true, nil
}
