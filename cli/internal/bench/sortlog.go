package bench

import (
	"os"
	"os/exec"

	"github.com/dopamine911/local-ai-lab/cli/internal/config"
)

// SortLog is `lab sort-log` (port of tuning/sort-log): writes the hand-sorted misses (tool-eval/classifications.json)
// to MLflow with tool-eval/sort_to_mlflow.py in the MLflow image. Safe to repeat.
func SortLog(cfg *config.Config) error {
	if err := os.Chdir(cfg.Tuning()); err != nil {
		return err
	}
	pwd, _ := os.Getwd()
	c := exec.Command("docker", "run", "--rm", "--network", "host", "-e", "GIT_PYTHON_REFRESH=quiet",
		"-v", pwd+":/tuning:ro", cfg.Images.MLflow, "python", "/tuning/tool-eval/sort_to_mlflow.py")
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return exitErr(exitCode(err))
	}
	return nil
}
