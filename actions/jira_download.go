package actions

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/fatih/color"
	"github.com/midN/jira-cloud-backuper/common"
	"gopkg.in/urfave/cli.v1"
)

// JiraDownload returns cli.Context related function
// which calls necessary JIRA APIs to download latest backup file
func JiraDownload() func(c *cli.Context) error {
	return func(c *cli.Context) error {
		filename := c.GlobalString("output")
		if filename == "" {
			filename = "jira.zip"
		}
		out, err := os.Create(filename)
		if err != nil {
			return common.CliError(err)
		}
		defer out.Close()

		client, host, err := common.AuthUser(c)
		if err != nil {
			return common.CliError(err)
		}

		latestID, err := latestJiraTaskID(client, host)
		if err != nil {
			return common.CliError(err)
		}

		downloadURL, err := common.JiraWaitForBackupReadyness(client, latestID, host)
		if err != nil {
			return common.CliError(err)
		}

		fmt.Println("Downloading to", filename)
		result, err := downloadLatestJira(client, downloadURL, out)
		if err != nil {
			return common.CliError(err)
		}

		fmt.Print(result)
		return nil
	}
}

func latestJiraTaskID(client http.Client, host string) (string, error) {
	url := host + "/rest/backup/1/export/lastTaskId"
	resp, _ := client.Get(url)
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New(string(body))
	}
	return string(body), nil
}

func downloadLatestJira(client http.Client, url string, out *os.File) (string, error) {
	resp, _ := client.Get(url)
	if resp.StatusCode == 404 {
		return "", errors.New("File not found at " + url)
	}
	defer resp.Body.Close()

	readerpt := &common.PassThru{Reader: resp.Body, Length: resp.ContentLength}
	count, err := io.Copy(out, readerpt)
	if err != nil {
		return "", err
	}

	return color.GreenString(fmt.Sprintln(
		"Download finished, file size:", count, "bytes.", "File:", out.Name())), nil
}
