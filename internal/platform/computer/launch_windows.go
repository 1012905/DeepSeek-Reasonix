package computer

import (
	"debug/pe"
	"path/filepath"
	"strings"
)

func nativeLaunchSpecification(app nativeApplication) (launchSpecification, error) {
	spec := launchSpecification{identity: app.Bundle}
	if strings.Contains(app.Bundle, "!") && !filepath.IsAbs(app.Bundle) {
		return spec, fail(CodeUnsupported, "packaged application launch requires prelaunch executable attestation")
	}
	file := strings.TrimSpace(app.LaunchPath)
	if strings.HasPrefix(file, `"`) && strings.HasSuffix(file, `"`) && len(file) >= 2 {
		file = file[1 : len(file)-1]
	}
	if !filepath.IsAbs(file) || strings.HasPrefix(file, `\\`) || !strings.EqualFold(filepath.Ext(file), ".exe") || !strings.EqualFold(file, app.Bundle) {
		return spec, fail(CodeUnsupported, "launch requires an exact local executable from the native catalogue without arguments")
	}
	spec.identity, spec.executable = file, file
	spec.args = map[string]any{"path": file}
	return spec, nil
}

func sameExecutablePath(actual, expected string) bool { return strings.EqualFold(actual, expected) }

func attestNativeLaunchBinary(file string) error {
	if !filepath.IsAbs(file) || strings.HasPrefix(file, `\\`) || !strings.EqualFold(filepath.Ext(file), ".exe") {
		return fail(CodeUnsupported, "resolved launch target must be a local executable path")
	}
	binary, err := pe.Open(file)
	if err != nil {
		return fail(CodeUnsupported, "launch requires a native PE executable")
	}
	defer binary.Close()
	if binary.Characteristics&pe.IMAGE_FILE_DLL != 0 || binary.OptionalHeader == nil {
		return fail(CodeUnsupported, "launch target is not an executable PE image")
	}
	return nil
}
