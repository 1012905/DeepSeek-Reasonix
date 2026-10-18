//go:build windows

package sandbox

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The launcher is an entry of whichever host binary imports this package: it
// reads its request from argv, runs the command under the restricted token, and
// exits with the command's status. Nothing else in the process runs.
func init() {
	l, argv, ok := parseWinLaunch(os.Args)
	if !ok {
		return
	}
	code, err := runRestricted(l, argv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reasonix sandbox:", err)
		os.Exit(126)
	}
	os.Exit(code)
}

var procCreateRestrictedToken = windows.NewLazySystemDLL("advapi32.dll").NewProc("CreateRestrictedToken")

const writeRestricted = 0x8

// tokenDefaultDACL is TOKEN_DEFAULT_DACL.
type tokenDefaultDACL struct{ defaultDACL *windows.ACL }

type sidAndAttributes struct {
	sid        *windows.SID
	attributes uint32
}

// restrictedToken is the process token narrowed to WRITE_RESTRICTED with the
// synthetic SID, the logon SID and Everyone as restricting SIDs, and lowered to
// Low integrity. A write needs the normal access check and one of those SIDs to
// pass, and only the roots carry the synthetic one. Everyone is there because a
// child with less cannot start.
func restrictedToken(sidText string) (windows.Token, error) {
	var cur windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ALL_ACCESS, &cur); err != nil {
		return 0, fmt.Errorf("open process token: %w", err)
	}
	defer cur.Close()
	synthetic, err := windows.StringToSid(sidText)
	if err != nil {
		return 0, err
	}
	logon, err := logonSID(cur)
	if err != nil {
		return 0, err
	}
	everyone, err := windows.CreateWellKnownSid(windows.WinWorldSid)
	if err != nil {
		return 0, err
	}
	system, _ := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	admins, _ := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	user, err := cur.GetTokenUser()
	if err != nil {
		return 0, err
	}
	list := []sidAndAttributes{{sid: synthetic}, {sid: logon}, {sid: everyone}}
	var out windows.Token
	r, _, e := procCreateRestrictedToken.Call(uintptr(cur), writeRestricted, 0, 0, 0, 0, uintptr(len(list)), uintptr(unsafe.Pointer(&list[0])), uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return 0, fmt.Errorf("create restricted token: %w", e)
	}
	// Objects the child creates take this DACL: without the restricting SIDs in
	// it the child could not open what it just made.
	if err := setDefaultDACL(out, synthetic, logon, everyone, system, admins, user.User.Sid); err != nil {
		out.Close()
		return 0, err
	}
	if err := setLowIntegrity(out); err != nil {
		out.Close()
		return 0, err
	}
	return out, nil
}

func logonSID(tok windows.Token) (*windows.SID, error) {
	buf := make([]byte, 512)
	var n uint32
	if err := windows.GetTokenInformation(tok, windows.TokenLogonSid, &buf[0], uint32(len(buf)), &n); err != nil {
		return nil, fmt.Errorf("logon SID: %w", err)
	}
	groups := (*windows.Tokengroups)(unsafe.Pointer(&buf[0]))
	if groups.GroupCount == 0 {
		return nil, fmt.Errorf("logon SID: the token has none (a session without an interactive logon cannot be restricted)")
	}
	return groups.AllGroups()[0].Sid.Copy()
}

func setDefaultDACL(tok windows.Token, sids ...*windows.SID) error {
	var entries []windows.EXPLICIT_ACCESS
	for _, s := range sids {
		if s == nil {
			continue
		}
		entries = append(entries, windows.EXPLICIT_ACCESS{
			AccessPermissions: windows.GENERIC_ALL, AccessMode: windows.GRANT_ACCESS, Inheritance: windows.NO_INHERITANCE,
			Trustee: windows.TRUSTEE{TrusteeForm: windows.TRUSTEE_IS_SID, TrusteeType: windows.TRUSTEE_IS_UNKNOWN, TrusteeValue: windows.TrusteeValueFromSID(s)},
		})
	}
	acl, err := windows.ACLFromEntries(entries, nil)
	if err != nil {
		return fmt.Errorf("default DACL: %w", err)
	}
	info := tokenDefaultDACL{defaultDACL: acl}
	if err := windows.SetTokenInformation(tok, windows.TokenDefaultDacl, (*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		return fmt.Errorf("set default DACL: %w", err)
	}
	return nil
}

// runRestricted starts argv suspended under the token, puts it in a job that
// ends the tree when the launcher does, resumes it, and returns its exit code.
// The command's standard handles are the launcher's own.
func runRestricted(l winLaunch, argv []string) (int, error) {
	tok, err := restrictedToken(l.SID)
	if err != nil {
		return 0, err
	}
	defer tok.Close()
	job, err := killOnCloseJob()
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(job)

	si := windows.StartupInfo{Flags: windows.STARTF_USESTDHANDLES}
	si.Cb = uint32(unsafe.Sizeof(si))
	for i, id := range []uint32{windows.STD_INPUT_HANDLE, windows.STD_OUTPUT_HANDLE, windows.STD_ERROR_HANDLE} {
		h, err := windows.GetStdHandle(id)
		if err != nil || h == 0 || h == windows.InvalidHandle {
			continue
		}
		_ = windows.SetHandleInformation(h, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT)
		switch i {
		case 0:
			si.StdInput = h
		case 1:
			si.StdOutput = h
		default:
			si.StdErr = h
		}
	}
	cmdline, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(argv))
	if err != nil {
		return 0, err
	}
	env := environmentBlock(mergeEnv(os.Environ(), l.Env))
	var pi windows.ProcessInformation
	if err := windows.CreateProcessAsUser(tok, nil, cmdline, nil, nil, true, windows.CREATE_SUSPENDED|windows.CREATE_UNICODE_ENVIRONMENT, &env[0], nil, &si, &pi); err != nil {
		return 0, fmt.Errorf("start %s: %w", argv[0], err)
	}
	defer windows.CloseHandle(pi.Process)
	defer windows.CloseHandle(pi.Thread)
	if err := windows.AssignProcessToJobObject(job, pi.Process); err != nil {
		_ = windows.TerminateProcess(pi.Process, 1)
		return 0, fmt.Errorf("assign to job: %w", err)
	}
	if _, err := windows.ResumeThread(pi.Thread); err != nil {
		_ = windows.TerminateProcess(pi.Process, 1)
		return 0, fmt.Errorf("resume: %w", err)
	}
	if _, err := windows.WaitForSingleObject(pi.Process, windows.INFINITE); err != nil {
		return 0, err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(pi.Process, &code); err != nil {
		return 0, err
	}
	return int(code), nil
}

func killOnCloseJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, fmt.Errorf("create job: %w", err)
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE}}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		windows.CloseHandle(job)
		return 0, fmt.Errorf("limit job: %w", err)
	}
	return job, nil
}

// environmentBlock is the NUL-separated, double-NUL-terminated UTF-16 form.
func environmentBlock(env []string) []uint16 {
	var block []uint16
	for _, kv := range env {
		u, err := windows.UTF16FromString(kv)
		if err != nil {
			continue
		}
		block = append(block, u...)
	}
	return append(block, 0)
}

// setLowIntegrity lowers the token to Low. Restricting SIDs are not consulted for
// a delete or a rename, so without this a restricted command could remove or move
// away anything the user may; mandatory integrity denies a Low writer both, except
// where the object is labelled Low, which the grant does for its roots.
func setLowIntegrity(tok windows.Token) error {
	sid, err := windows.StringToSid(lowIntegritySID)
	if err != nil {
		return err
	}
	tml := windows.Tokenmandatorylabel{Label: windows.SIDAndAttributes{Sid: sid, Attributes: windows.SE_GROUP_INTEGRITY}}
	if err := windows.SetTokenInformation(tok, windows.TokenIntegrityLevel, (*byte)(unsafe.Pointer(&tml)), tml.Size()); err != nil {
		return fmt.Errorf("set integrity level: %w", err)
	}
	return nil
}
