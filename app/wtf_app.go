package app

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/gdamore/tcell/terminfo/extended"
	"github.com/gdamore/tcell/v2"
	"github.com/olebedev/config"
	"github.com/radovskyb/watcher"
	"github.com/rivo/tview"

	"github.com/wtfutil/wtf/cfg"
	logger "github.com/wtfutil/wtf/logger"
	"github.com/wtfutil/wtf/utils"
	"github.com/wtfutil/wtf/wtf"
)

// WtfApp is the container for a collection of widgets that are all constructed from a single
// configuration file and displayed together
type WtfApp struct {
	TViewApp *tview.Application

	config         *config.Config
	configFilePath string
	pages          *tview.Pages
	validator      *ModuleValidator

	screens []WtfScreen

	currentScreen *WtfScreen

	// The redrawChan channel is used to allow modules to signal back to the main loop that
	// the screen needs to be explicitly redrawn, instead of waiting for tcell to redraw
	// on a user event, because something has visually changed
	redrawChan chan bool
}

type WtfScreen struct {
	title        string
	index        int
	widgets      []wtf.Wtfable
	display      *Display
	focusTracker FocusTracker
	mnemonic     string
}

// NewWtfApp creates and returns an instance of WtfApp
func NewWtfApp(tviewApp *tview.Application, config *config.Config, configFilePath string) *WtfApp {

	tview.Borders.TopLeftFocus = tview.BoxDrawingsLightArcDownAndRight
	tview.Borders.TopRightFocus = tview.BoxDrawingsLightArcDownAndLeft
	tview.Borders.BottomLeftFocus = tview.BoxDrawingsLightArcUpAndRight
	tview.Borders.BottomRightFocus = tview.BoxDrawingsLightArcUpAndLeft

	tview.Borders.TopLeft = tview.BoxDrawingsLightArcDownAndRight
	tview.Borders.TopRight = tview.BoxDrawingsLightArcDownAndLeft
	tview.Borders.BottomLeft = tview.BoxDrawingsLightArcUpAndRight
	tview.Borders.BottomRight = tview.BoxDrawingsLightArcUpAndLeft

	wtfApp := &WtfApp{
		TViewApp: tviewApp,

		config:         config,
		configFilePath: configFilePath,
		pages:          tview.NewPages(),
		screens:        []WtfScreen{},

		redrawChan: make(chan bool, 1),
	}

	wtfApp.TViewApp.SetBeforeDrawFunc(func(s tcell.Screen) bool {
		s.Clear()
		return false
	})

	screens, err := config.List("wtf.screens")
	if err != nil {
		log.Println("screens: ", err)
	}

	for _, ss := range screens {
		screen := ss.(map[string]interface{})
		s := WtfScreen{
			title: screen["title"].(string),
			index: screen["index"].(int),
		}

		wtfApp.screens = append(wtfApp.screens, s)
	}

	wtfApp.initPage(1)

	firstWidget := wtfApp.currentScreen.widgets[0]
	wtfApp.pages.Box.SetBackgroundColor(
		wtf.ColorFor(
			firstWidget.CommonSettings().Colors.WidgetTheme.Background,
		),
	)

	wtfApp.TViewApp.SetRoot(wtfApp.pages, true)

	wtfApp.TViewApp.SetInputCapture(wtfApp.keyboardIntercept)

	go handleRedraws(wtfApp.TViewApp, wtfApp.redrawChan)

	return wtfApp
}

func (wtfApp *WtfApp) getScreenByIndex(index int) *WtfScreen {
	for _, screen := range wtfApp.screens {
		if screen.index == index {
			return &screen
		}
	}
	return nil
}

func (wtfApp *WtfApp) initPage(index int) {

	newScreen := wtfApp.getScreenByIndex(index)

	if newScreen == nil {
		return
	}

	currentScreen := wtfApp.currentScreen
	if currentScreen != nil {
		currentScreen.stopAllWidgets()
		wtfApp.pages.RemovePage("grid" + strconv.Itoa(currentScreen.index))
	}

	wtfApp.currentScreen = newScreen

	wtfApp.currentScreen.widgets = MakeWidgets(wtfApp)
	wtfApp.currentScreen.display = NewDisplay(wtfApp)
	wtfApp.currentScreen.focusTracker = NewFocusTracker(wtfApp.TViewApp, wtfApp.currentScreen.widgets, wtfApp.config)
	wtfApp.validator = NewModuleValidator()

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(wtfApp.currentScreen.display.TabBar, 1, 0, false).
		AddItem(wtfApp.currentScreen.display.Grid, 0, 10, true)

	wtfApp.pages.AddPage("grid"+strconv.Itoa(wtfApp.currentScreen.index), flex, true, true)

	wtfApp.pages.SwitchToPage("grid" + strconv.Itoa(wtfApp.currentScreen.index))

	wtfApp.TViewApp.SetRoot(wtfApp.pages, true)

	wtfApp.validator.Validate(wtfApp.currentScreen.widgets)

	// Create a watcher to handle calls to redraw the screen
	go wtfApp.scheduleWidgets()
}

func (wtfApp *WtfApp) changeScreen(screenIndex int) {
	logger.Log(fmt.Sprintf("Changing to screen: %d", screenIndex))
	wtfApp.initPage(screenIndex)

}

func handleRedraws(tviewApp *tview.Application, redrawChan chan bool) {
	if redrawChan == nil {
		return
	}

	for {
		data, ok := <-redrawChan
		if !ok {
			return
		}

		if data {
			tviewApp.Draw()
		}
	}
}

/* -------------------- Exported Functions -------------------- */

// Exit quits the app
func (wtfApp *WtfApp) Exit() {
	wtfApp.Stop()
	wtfApp.TViewApp.Stop()
	wtfApp.DisplayExitMessage()
	os.Exit(0)
}

// Execute starts the underlying tview app
func (wtfApp *WtfApp) Execute() error {
	if err := wtfApp.TViewApp.Run(); err != nil {
		return err
	}

	return nil
}

// Start initializes the app
func (wtfApp *WtfApp) Start() {
	//go wtfApp.scheduleWidgets()
	go wtfApp.watchForConfigChanges()
}

// Stop kills all the currently-running widgets in this app
func (wtfApp *WtfApp) Stop() {
	wtfApp.currentScreen.stopAllWidgets()
	close(wtfApp.redrawChan)
}

/* -------------------- Unexported Functions -------------------- */

func (wtfScreen *WtfScreen) stopAllWidgets() {
	if wtfScreen == nil {
		return
	}
	for _, widget := range wtfScreen.widgets {
		widget.Stop()
	}
}

func (wtfApp *WtfApp) keyboardIntercept(event *tcell.EventKey) *tcell.EventKey {

	logger.Log(fmt.Sprintf("Key: %d", event.Key()))

	// These keys are global keys used by the app. Widgets should not implement these keys
	switch event.Key() {
	case tcell.KeyCtrlC:
		wtfApp.Stop()
		wtfApp.TViewApp.Stop()
		wtfApp.DisplayExitMessage()
	case tcell.KeyCtrlR:
		wtfApp.refreshAllWidgets()
		return nil
	case tcell.KeyCtrlSpace:
		// FIXME: This can't reside in the app, the app doesn't know about
		// the AppManager. The AppManager needs to catch this one
		fmt.Println("Next app")
		return nil
	case tcell.KeyTab:
		wtfApp.currentScreen.focusTracker.Next()
	case tcell.KeyBacktab:
		wtfApp.currentScreen.focusTracker.Prev()
		return nil
	case tcell.KeyEsc:
		wtfApp.currentScreen.focusTracker.None()
	}

	// check if the key is a number and ctrl
	logger.Log(fmt.Sprintf("Key: %d", event.Key()))
	logger.Log(fmt.Sprintf("Modifier: %d", event.Modifiers()))
	if event.Rune() >= '0' && event.Rune() <= '9' {
		wtfApp.changeScreen(int(event.Rune() - '0'))
	}

	// Checks to see if any widget has been assigned the pressed key as its focus key
	//if wtfApp.currentScreen.focusTracker.FocusOn(string(event.Rune())) {
	//	return nil
	//}

	// If no specific widget has focus, then allow the key presses to fall through to the app
	if !wtfApp.currentScreen.focusTracker.IsFocused {
		switch string(event.Rune()) {
		case "q":
			wtfApp.Exit()
		case "/":
			return nil
		default:
		}
	}

	return event
}

func (wtfApp *WtfApp) refreshAllWidgets() {
	for _, widget := range wtfApp.currentScreen.widgets {
		go widget.Refresh()
	}
}

func (wtfApp *WtfApp) scheduleWidgets() {
	for _, widget := range wtfApp.currentScreen.widgets {
		go Schedule(widget)
	}
}

func (wtfApp *WtfApp) watchForConfigChanges() {
	watch := watcher.New()

	// Notify write events
	watch.FilterOps(watcher.Write)

	go func() {
		for {
			select {
			case <-watch.Event:
				wtfApp.Stop()

				config := cfg.LoadWtfConfigFile(wtfApp.configFilePath)
				newApp := NewWtfApp(wtfApp.TViewApp, config, wtfApp.configFilePath)
				openURLUtil := utils.ToStrs(config.UList("wtf.openUrlUtil", []interface{}{}))
				utils.Init(config.UString("wtf.openFileUtil", "open"), openURLUtil)

				newApp.Start()
			case err := <-watch.Error:
				if err == watcher.ErrWatchedFileDeleted {
					// Usually happens because the watcher looks for the file as the OS is updating it
					continue
				}
				log.Fatalln(err)
			case <-watch.Closed:
				return
			}
		}
	}()

	// Watch config file for changes.
	absPath, _ := utils.ExpandHomeDir(wtfApp.configFilePath)
	if err := watch.Add(absPath); err != nil {
		log.Fatalln(err)
	}

	// Start the watching process - it'll check for changes every 100ms.
	if err := watch.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}
