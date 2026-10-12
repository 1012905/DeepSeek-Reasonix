package control

import "reasonix/internal/contract/event"

func (c *Controller) ExtensionSurfaces() []event.ExtensionSurfacePayload {
	if h := c.extensionUIHub(); h != nil {
		return h.Surfaces()
	}
	return nil
}
