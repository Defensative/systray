/*
Package systray is a cross platfrom Go library to place an icon and menu in the notification area.
Supports Windows, Mac OSX and Linux currently.
Methods can be called from any goroutine except Run(), which should be called at the very beginning of main() to lock at main thread.
*/
package systray

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/getlantern/golog"
)

// MenuItem is used to keep track each menu item of systray
// Don't create it directly, use the one systray.AddMenuItem() returned
type MenuItem struct {
	// id uniquely identify a menu item, not supposed to be modified
	id int32
	// title is the text shown on menu item
	title string
	// tooltip is the text shown when pointing to menu item
	tooltip string
	// disabled menu item is grayed out and has no effect when clicked
	disabled bool
	// checked menu item has a tick before the title
	checked bool
	// indicates should be removed
	remove bool
	// indicates should be a separator
	separator bool
}

var (
	log = golog.LoggerFor("systray")

	ClickedCh     = make(chan *MenuItem)
	readyCh       = make(chan interface{})
	menuItems     = make(map[int32]*MenuItem)
	menuItemsLock sync.RWMutex

	currentID int32
)

// Run initializes GUI and starts the event loop, then invokes the onReady
// callback.
// It blocks until systray.Quit() is called.
// Should be called at the very beginning of main() to lock at main thread.
func Run(onReady func()) {
	runtime.LockOSThread()
	go func() {
		<-readyCh
		onReady()
	}()

	nativeLoop()
}

// Quit the systray
func Quit() {
	quit()
}

// AddMenuItem adds menu item with designated title and tooltip, returning a channel
// that notifies whenever that menu item is clicked.
//
// It can be safely invoked from different goroutines.
func AddMenuItem(title string, tooltip string, before *MenuItem) *MenuItem {
	id := atomic.AddInt32(&currentID, 1)
	item := &MenuItem{id, title, tooltip, false, false, false, false}
	item.update(before)
	return item
}

// SetTitle set the text to display on a menu item
func (item *MenuItem) SetTitle(title string) {
	item.title = title
	item.update(nil)
}

// SetTitle set the text to display on a menu item
func (item *MenuItem) GetTitle() (string) {
	return item.title
}

// SetTooltip set the tooltip to show when mouse hover
func (item *MenuItem) SetTooltip(tooltip string) {
	item.tooltip = tooltip
	item.update(nil)
}

// SetTooltip set the tooltip to show when mouse hover
func (item *MenuItem) SetSeparator(s bool) {
	item.separator = s
	item.update(nil)
}

// Disabled checkes if the menu item is disabled
func (item *MenuItem) Disabled() bool {
	return item.disabled
}

// Enable a menu item regardless if it's previously enabled or not
func (item *MenuItem) Enable() {
	item.disabled = false
	item.update(nil)
}

// Disable a menu item regardless if it's previously disabled or not
func (item *MenuItem) Disable() {
	item.disabled = true
	item.update(nil)
}

// Checked returns if the menu item has a check mark
func (item *MenuItem) Checked() bool {
	return item.checked
}

// Check a menu item regardless if it's previously checked or not
func (item *MenuItem) Check() {
	item.checked = true
	item.update(nil)
}

// Uncheck a menu item regardless if it's previously unchecked or not
func (item *MenuItem) Uncheck() {
	item.checked = false
	item.update(nil)
}

// Remove a menu item
//  * Currently implimented on Windows only
func (item *MenuItem) Remove() {
	item.remove = true
	item.update(nil)
}

// update propogates changes on a menu item to systray
func (item *MenuItem) update(before *MenuItem) {
	menuItemsLock.Lock()
	defer menuItemsLock.Unlock()
	menuItems[item.id] = item
	addOrUpdateMenuItem(item, before)
}

func systrayReady() {
	readyCh <- nil
}

func systrayMenuItemSelected(id int32) {
	menuItemsLock.RLock()
	item := menuItems[id]
	menuItemsLock.RUnlock()
	select {
	case ClickedCh <- item:
	// in case no one waiting for the channel
	default:
	}
}
