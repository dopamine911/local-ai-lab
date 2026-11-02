// Package machines: commands that set up or look after a machine (issue #6 step 5): models, backup, setup check.
package machines

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// ModelsDir is where every machine keeps its model files.
const ModelsDir = "/srv/models"

// Models is `lab models [machine] [filter]` (port of setup/models.sh): download every model listed in
// setup/models-<machine>.txt into /srv/models (only lines whose destination contains filter), then check each file's
// sha256. Finished files are skipped, partial ones resume (wget -c). Returns false if any download or check failed.
// Unlike the script (always exit 0), lab exits 1 then, so agents and scripts can tell.
func Models(cfg *config.Config, machine, filter string) (bool, error) {
	if machine == "" {
		machine, _ = os.Hostname()
	}
	list := filepath.Join(cfg.Repo, "setup", "models-"+machine+".txt")
	b, err := os.ReadFile(list)
	if err != nil {
		fmt.Println("no list " + list)
		return false, nil
	}
	if err := os.Chdir(ModelsDir); err != nil {
		return false, err
	}
	ok := true
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 0 || strings.HasPrefix(f[0], "#") {
			continue
		}
		for len(f) < 6 {
			f = append(f, "")
		}
		repo, rev, path, dest, sizeS, sha := f[0], f[1], f[2], f[3], f[4], strings.Join(f[5:], " ")
		if filter != "" && !strings.Contains(dest, filter) {
			continue
		}
		os.MkdirAll(filepath.Dir(dest), 0o755)
		size, _ := strconv.ParseInt(sizeS, 10, 64)
		if st, err := os.Stat(dest); err == nil && st.Mode().IsRegular() && st.Size() == size {
			fmt.Println("have " + dest)
		} else {
			fmt.Printf("%s downloading %s (%d MB)\n", now(), dest, size/1000000)
			w := exec.Command("wget", "-q", "-c", "-O", dest, "https://huggingface.co/"+repo+"/resolve/"+rev+"/"+path)
			w.Stdout, w.Stderr = os.Stdout, os.Stderr
			if err := w.Run(); err != nil {
				fmt.Println("FAILED download " + dest)
				ok = false
				continue
			}
		}
		fmt.Printf("%s checking %s\n", now(), dest)
		if got, err := fileSHA256(dest); err != nil || got != sha {
			fmt.Println(dest + ": FAILED") // what sha256sum -c --quiet printed
			fmt.Println("CHECKSUM FAILED " + dest)
			ok = false
		}
	}
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, "# UNKNOWN") {
			fmt.Println("not in the list, get by hand: " + l)
		}
	}
	if ok {
		fmt.Println("MODELS-OK")
	} else {
		fmt.Println("MODELS-FAILED")
	}
	return ok, nil
}

func now() string { return time.Now().Format("15:04:05") }

func fileSHA256(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
