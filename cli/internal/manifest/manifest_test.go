package manifest

import (
	"os"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Expected values are what tuning/manifest.sh printed for the same outputs (2026-11-02).
func TestParsers(t *testing.T) {
	cases := []struct{ name, got, want string }{
		{"llama.cpp commit", LlamaCppCommit(fixture(t, "zaphod-llama-server-version.txt")), "0ec4027"},
		{"llama-swap version", LlamaSwapVersion(fixture(t, "zaphod-llama-swap-version.txt")), "v268"},
		{"zaphod vulkan devices", VulkanDevices(fixture(t, "zaphod-list-devices.txt")),
			"Vulkan0: AMD Radeon AI PRO R9700 (RADV GFX1201) (32768 MiB); Vulkan1: AMD Radeon AI PRO R9700 (RADV GFX1201) (32768 MiB)"},
		{"marvin vulkan devices", VulkanDevices(fixture(t, "marvin-list-devices.txt")),
			"Vulkan0: Radeon 8060S Graphics (RADV STRIX_HALO) (123392 MiB)"},
		{"no commit", LlamaCppCommit("error: no such file\n"), ""},
		{"no devices", VulkanDevices("Available devices:\n"), ""},
		{"cpu", cpuModel("processor\t: 0\nmodel name\t: AMD Ryzen 9 3900X 12-Core Processor\n"), "AMD Ryzen 9 3900X 12-Core Processor"},
		{"ram", ramGB("MemTotal:       63335548 kB\nMemFree: 1 kB\n"), "60"},
		{"image id", imageID("sha256:cca61f141d10367031cbfaaff969e975bf78e3869bddfaf7fb8a11f57900e8f2\n"), "cca61f141d10"},
		{"unknown", orUnknown("\n"), "unknown"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, c.got, c.want)
		}
	}
}
