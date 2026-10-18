//go:build windows

package sandbox

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// grantMask is FILE_ALL_ACCESS: the synthetic SID is the write bound itself, so
// it carries everything on the roots it is granted and nothing anywhere else.
const grantMask = windows.ACCESS_MASK(0x1F01FF)

// WindowsGrantOps edits real DACLs. Granting and revoking propagate to the
// children of a root, which costs time proportional to the tree (about 0.7 s
// per 20,000 files measured), so the book grants a workspace once.
func WindowsGrantOps() grantOps { return daclOps{} }

type daclOps struct{}

func (daclOps) Grant(root, sid string) error {
	if err := editDACL(root, sid, windows.GRANT_ACCESS); err != nil {
		return err
	}
	return setRootLabel(root, lowLabelSDDL)
}

func (daclOps) Revoke(root, sid string) error {
	if err := editDACL(root, sid, windows.REVOKE_ACCESS); err != nil {
		return err
	}
	return setRootLabel(root, "S:")
}

func editDACL(root, sidText string, mode windows.ACCESS_MODE) error {
	sid, err := windows.StringToSid(sidText)
	if err != nil {
		return fmt.Errorf("sandbox: parse SID: %w", err)
	}
	sd, err := windows.GetNamedSecurityInfo(root, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("sandbox: read DACL of %s: %w", root, err)
	}
	old, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("sandbox: read DACL of %s: %w", root, err)
	}
	entry := windows.EXPLICIT_ACCESS{
		AccessPermissions: grantMask,
		AccessMode:        mode,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee:           windows.TRUSTEE{TrusteeForm: windows.TRUSTEE_IS_SID, TrusteeType: windows.TRUSTEE_IS_UNKNOWN, TrusteeValue: windows.TrusteeValueFromSID(sid)},
	}
	next, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{entry}, old)
	if err != nil {
		return fmt.Errorf("sandbox: build DACL for %s: %w", root, err)
	}
	if err := windows.SetNamedSecurityInfo(root, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, next, nil); err != nil {
		return fmt.Errorf("sandbox: write DACL of %s: %w", root, err)
	}
	return nil
}

// lowIntegritySID and lowLabelSDDL: the label that lets a Low-integrity writer
// write into a root, inherited by what is created in it later.
const (
	lowIntegritySID = "S-1-16-4096"
	lowLabelSDDL    = "S:(ML;OICI;NW;;;LW)"
	labelAceType    = 0x11 // SYSTEM_MANDATORY_LABEL_ACE_TYPE
)

// setRootLabel replaces the mandatory label of root with the one sddl describes.
func setRootLabel(root, sddl string) error {
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return err
	}
	sacl, _, err := sd.SACL()
	if err != nil {
		return err
	}
	if err := windows.SetNamedSecurityInfo(root, windows.SE_FILE_OBJECT, windows.LABEL_SECURITY_INFORMATION, nil, nil, nil, sacl); err != nil {
		return fmt.Errorf("sandbox: write the integrity label of %s: %w", root, err)
	}
	return nil
}

// hasLowLabel reports whether root carries the Low label.
func hasLowLabel(root string) (bool, error) {
	sd, err := windows.GetNamedSecurityInfo(root, windows.SE_FILE_OBJECT, windows.LABEL_SECURITY_INFORMATION)
	if err != nil {
		return false, err
	}
	if ctrl, _, err := sd.Control(); err != nil || ctrl&windows.SE_SACL_PRESENT == 0 {
		return false, err
	}
	sacl, _, err := sd.SACL()
	if err != nil || sacl == nil {
		return false, err
	}
	low, err := windows.StringToSid(lowIntegritySID)
	if err != nil {
		return false, err
	}
	for i := uint32(0); i < uint32(sacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(sacl, i, &ace); err != nil {
			return false, err
		}
		if ace.Header.AceType == labelAceType && windows.EqualSid(low, (*windows.SID)(unsafe.Pointer(&ace.SidStart))) {
			return true, nil
		}
	}
	return false, nil
}

// Granted reports whether root has both halves of a grant: an allow entry for
// the SID and the Low label.
func (daclOps) Granted(root, sidText string) (bool, error) {
	if ok, err := hasLowLabel(root); err != nil || !ok {
		return false, err
	}
	sid, err := windows.StringToSid(sidText)
	if err != nil {
		return false, err
	}
	sd, err := windows.GetNamedSecurityInfo(root, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return false, err
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil {
		return false, err
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(acl, i, &ace); err != nil {
			return false, err
		}
		if ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE && windows.EqualSid(sid, (*windows.SID)(unsafe.Pointer(&ace.SidStart))) {
			return true, nil
		}
	}
	return false, nil
}
