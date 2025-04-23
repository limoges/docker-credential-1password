package onepassword

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/tidwall/gjson"
)

var (
	ErrURLMismatch = errors.New("field url doesn't match title")
)

type Helper struct {
	vault    string
	category string
	tag      string
	debug    io.Writer
}

func NewHelper() *Helper {
	debug := io.Discard
	_, debugEnabled := os.LookupEnv("DOCKER_CREDENTIAL_1PASSWORD_DEBUG")
	if debugEnabled {
		debug = os.Stderr
	}
	vault := os.Getenv("DOCKER_CREDENTIAL_1PASSWORD_VAULT")
	if vault == "" {
		vault = "Docker"
	}
	category := os.Getenv("DOCKER_CREDENTIAL_1PASSWORD_CATEGORY")
	if category == "" {
		category = "Server"
	}
	tag := os.Getenv("DOCKER_CREDENTIAL_1PASSWORD_TAG")
	if tag == "" {
		tag = credentials.CredsLabel
	}
	h := &Helper{}
	h.vault = vault
	h.category = category
	h.tag = tag
	h.debug = debug
	return h
}

func (h *Helper) Add(creds *credentials.Credentials) error {
	cmd := h.itemCreate(creds.ServerURL, creds.Username, creds.Secret)
	fmt.Fprintln(h.debug, cmd.String())
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (h *Helper) Delete(serverURL string) error {
	cmd := h.itemDelete(serverURL)
	cmd.Stderr = os.Stderr
	fmt.Fprintln(h.debug, cmd.String())
	return cmd.Run()
}

func (h *Helper) Get(serverURL string) (username, password string, err error) {
	var output bytes.Buffer
	cmd := h.itemGet(serverURL)
	cmd.Stdout = &output
	cmd.Stderr = os.Stderr
	fmt.Fprintln(h.debug, cmd.String())
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("failed to run command: %w", err)
	}
	parsed := gjson.ParseBytes(output.Bytes())
	url := parsed.Get(`fields.#(id=="url").value`).String()
	if url != serverURL {
		return "", "", fmt.Errorf("%q doesn't match %q: %w", url, serverURL, ErrURLMismatch)
	}
	username = parsed.Get(`fields.#(id=="username").value`).String()
	password = parsed.Get(`fields.#(id=="password").value`).String()
	return username, password, nil
}

func (h *Helper) List() (map[string]string, error) {
	var (
		listOutput bytes.Buffer
		getOutput  bytes.Buffer
	)
	listCmd := h.itemList()
	listCmd.Stdout = &listOutput
	listCmd.Stderr = os.Stderr
	fmt.Fprintln(h.debug, listCmd.String())
	if err := listCmd.Run(); err != nil {
		return nil, err
	}
	getCmd := h.itemGetStdin()
	getCmd.Stdin = &listOutput
	getCmd.Stdout = &getOutput
	getCmd.Stderr = os.Stderr
	fmt.Fprintln(h.debug, getCmd.String())
	if err := getCmd.Run(); err != nil {
		return nil, err
	}
	credentials := make(map[string]string)
	if err := parseOutput(&getOutput, credentials); err != nil {
		return nil, err
	}
	return credentials, nil
}

func parseOutput(r io.Reader, credentials map[string]string) error {
	reader := bufio.NewReader(r)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to readline: %w", err)
		}
		serverURL, username, err := parseLine(line)
		if err != nil {
			return fmt.Errorf("failed to parse line: %w", err)
		}
		credentials[serverURL] = username
	}
}

func parseLine(line []byte) (serverURL, username string, err error) {
	values := strings.Split(string(line), ",")
	if len(values) < 2 {
		return "", "", errors.New("malformed line")
	}
	return values[0], values[1], nil
}

type cmd struct {
	vault string
}

func newCmd(vault string) *cmd {
	c := &cmd{}
	c.vault = vault
	return c
}

func (h *Helper) itemList() *exec.Cmd {
	return exec.Command(
		"op",
		"item",
		"list",
		fmt.Sprintf("--vault=%s", h.vault),
		fmt.Sprintf("--categories=%s", h.category),
		fmt.Sprintf("--tags=%s", h.tag),
		"--format=json",
	)
}

func (h *Helper) itemGetStdin() *exec.Cmd {
	return exec.Command(
		"op",
		"item",
		"get",
		"-",
		"--fields=url,username",
	)
}

func (h *Helper) itemGet(serverURL string) *exec.Cmd {
	return exec.Command(
		"op",
		"item",
		"get",
		"--format=json",
		fmt.Sprintf("--vault=%s", h.vault),
		serverURL,
	)
}

func (h *Helper) itemCreate(serverURL, username, secret string) *exec.Cmd {
	return exec.Command(
		"op",
		"item",
		"create",
		"--format=json",
		fmt.Sprintf("--category=%s", h.category),
		fmt.Sprintf("--vault=%s", h.vault),
		fmt.Sprintf("--title=%s", serverURL),
		fmt.Sprintf("--tag=%s", h.tag),
		"-",
		fmt.Sprintf("username=%s", username),
		fmt.Sprintf("password=%s", secret),
		fmt.Sprintf("URL=%s", serverURL),
	)
}

func (h *Helper) itemDelete(serverURL string) *exec.Cmd {
	return exec.Command(
		"op",
		"item",
		"delete",
		"--archive",
		fmt.Sprintf("--vault=%s", h.vault),
		serverURL,
	)
}
