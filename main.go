// termsuji is a application to play Go in a terminal: against GNU Go on this
// computer, or on online-go.com. The online part is not complete and can
// only read board state and play moves, and is intended more as a reference
// than a full-fledged application.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/gdamore/tcell/v2"
	"github.com/lvank/termsuji/api"
	"github.com/lvank/termsuji/config"
	"github.com/lvank/termsuji/engine"
	"github.com/lvank/termsuji/ui"
	"github.com/rivo/tview"
)

var lastRefresh time.Time = time.Now()
var app *tview.Application
var rootPage *tview.Pages
var gameListFrame *tview.Frame
var gameList *tview.List
var frameHint *tview.Frame
var gameBoard *ui.GoBoardUI
var setLoading func(bool)
var menu *tview.List
var menuFrame *tview.Frame

// settingsName is where a local game's settings are kept, beside the game.
const settingsName = "game.json"

// dataDir is the folder local games are saved in, made as needed.
func dataDir() (string, error) {
	dir := filepath.Join(xdg.DataHome, "termsuji")
	return dir, os.MkdirAll(dir, 0o755)
}

func main() {
	cfg, err := config.InitConfig()
	if err != nil {
		panic(err)
	}
	cfg.Save() // TODO settings screen or something
	app = tview.NewApplication()
	rootPage = tview.NewPages()
	rootPage.SetBorder(true).SetTitle("termsuji")
	gameList = tview.NewList()
	gameListFrame = tview.NewFrame(gameList)
	gameListFrame.SetBorders(0, 0, 0, 0, 0, 0)

	loadingModal := tview.NewModal()
	loadingModal.SetText("Loading...")
	setLoading = func(loading bool) {
		f := rootPage.ShowPage
		if !loading {
			f = rootPage.HidePage
		}
		f("loading")
	}
	gameFrame := tview.NewFlex()
	gameHint := tview.NewTextView()
	gameHint.SetBorder(true)
	gameBoard = ui.NewGoBoard(app, cfg, gameHint)
	gameFrame.
		AddItem(gameBoard.Box, 0, 3, true).
		AddItem(gameHint, 0, 2, false)
	newGameForm := tview.NewForm()
	gameBoard.Box.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == 'q' {
			if gameBoard.SelectedTile() != nil {
				gameBoard.ResetSelection()
			} else if gameBoard.LocalGame() {
				gameBoard.Close()
				showMenu()
			} else {
				gameBoard.Close()
				async(func() {
					refreshGames()
					rootPage.SwitchToPage("browser")
				})
			}
			return nil
		}
		switch event.Key() {
		case tcell.KeyUp:
			gameBoard.MoveSelection(0, -1)
		case tcell.KeyDown:
			gameBoard.MoveSelection(0, 1)
		case tcell.KeyLeft:
			gameBoard.MoveSelection(-1, 0)
		case tcell.KeyRight:
			gameBoard.MoveSelection(1, 0)
		case tcell.KeyEnter:
			selTile := gameBoard.SelectedTile()
			if selTile == nil {
				return nil
			}
			gameBoard.PlayMove(selTile.X, selTile.Y)
		case tcell.KeyRune:
			switch event.Rune() {
			case 'p':
				gameBoard.PlayMove(-1, -1)
			case 't':
				rootPage.ShowPage("themes")
			case 'u':
				gameBoard.UndoLocal()
			case 'n':
				if gameBoard.LocalGame() && gameBoard.BoardState.Finished() {
					gameBoard.Close()
					rootPage.SwitchToPage("newgame")
					app.SetFocus(newGameForm)
				}
			}
		}
		return event
	})
	gameList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				showMenu()
				return nil
			case 'r':
				async(func() {
					refreshGames()
				})
				return nil
			case 't':
				rootPage.ShowPage("themes")
				return nil
			}
		}
		return event
	})

	auth := config.InitAuthData()
	loginForm := tview.NewForm()
	loginFrame := tview.NewFrame(loginForm)
	loginForm.
		AddInputField("Username", auth.Username, 32, nil, nil). //if we have a cached username, prefill it
		AddPasswordField("Password", "", 32, '*', nil).
		AddButton("Submit", func() {
			err := api.AuthenticatePassword(
				loginForm.GetFormItem(0).(*tview.InputField).GetText(),
				loginForm.GetFormItem(1).(*tview.InputField).GetText(),
			)
			if err != nil {
				loginFrame.Clear().AddText(err.Error(), true, tview.AlignLeft, tcell.PaletteColor(1))
				return
			}
			storeAuthData(auth)
			refreshGames()
			rootPage.SwitchToPage("browser")
		}).
		AddButton("Back", showMenu)
	loginFrame.
		SetBorders(0, 0, 0, 0, 1, 0).
		AddText("Log in to OGS", true, tview.AlignLeft, tcell.PaletteColor(3))

	themeList := tview.NewList()
	themeList.SetTitle("Choose a theme")
	selectTheme := func(i int, main, secondary string, shortcut rune) {
		var theme config.Theme
		switch main {
		case "classic":
			theme = config.ClassicTheme
		case "default":
			theme = config.DefaultTheme
		case "vaporwave":
			theme = config.VaporwaveTheme
		case "unicode":
			theme = config.UnicodeTheme
		case "catdog":
			theme = config.CatdogTheme
		case "hongoku":
			theme = config.HongokuTheme
		default: //No action
		}

		if main != "quit" {
			cfg.Theme = theme
			cfg.Save()
			gameBoard.SetConfig(cfg)
		}
		rootPage.HidePage("themes")
	}
	themeList.
		AddItem("classic", "Lines and points on wood (the default)", '0', nil).
		AddItem("default", "Coloured squares (termsuji's original look)", '1', nil).
		AddItem("vaporwave", "Magenta/cyan board theme", '2', nil).
		AddItem("unicode", "Display board with Unicode emoji symbols", '3', nil).
		AddItem("catdog", "Board becomes zoo", '4', nil).
		AddItem("hongoku", "Board becomes unreadable", '5', nil).
		AddItem("quit", "Keep your current settings", 'q', nil).
		SetSelectedFunc(selectTheme)

	//a new game against GNU Go
	sizes := []int{9, 13, 19}
	levels := []string{"1 (weakest)", "2", "3", "4", "5", "6", "7", "8", "9", "10 (strongest)"}
	handicaps := []string{"none", "2", "3", "4", "5", "6", "7", "8", "9"}
	settings := ui.LocalSettings{Size: 9, Human: engine.Black, Level: 5}
	newGameForm.
		AddDropDown("Board", []string{"9x9", "13x13", "19x19"}, 0, func(_ string, i int) {
			if i >= 0 {
				settings.Size = sizes[i]
			}
		}).
		AddDropDown("You play", []string{"Black (first)", "White"}, 0, func(_ string, i int) {
			settings.Human = engine.Black
			if i == 1 {
				settings.Human = engine.White
			}
		}).
		AddDropDown("GNU Go's strength", levels, 4, func(_ string, i int) {
			settings.Level = i + 1
		}).
		AddDropDown("Handicap stones for Black", handicaps, 0, func(_ string, i int) {
			settings.Handicap = 0
			if i > 0 {
				settings.Handicap = i + 1
			}
		}).
		AddButton("Play", func() {
			s := settings
			//komi: compensation for White moving second; half a point
			//with handicap stones, which make up for more than that
			s.Komi = 6.5
			if s.Handicap > 0 {
				s.Komi = 0.5
			}
			startLocal(s, false)
		}).
		AddButton("Back", showMenu)
	newGameFrame := tview.NewFrame(newGameForm).
		SetBorders(0, 0, 1, 0, 1, 0).
		AddText("A new game against GNU Go", true, tview.AlignLeft, tcell.PaletteColor(3)).
		AddText("Up/Down or Tab: choose, Return: open a list, Esc: back", false, tview.AlignLeft, tcell.ColorDefault)
	newGameForm.SetCancelFunc(showMenu)

	menu = tview.NewList()
	menuFrame = tview.NewFrame(menu).SetBorders(1, 0, 1, 0, 2, 0)

	errorModal := tview.NewModal().AddButtons([]string{"OK"}).
		SetDoneFunc(func(int, string) { rootPage.HidePage("error") })

	rootPage.AddPage("menu", menuFrame, true, true)
	rootPage.AddPage("newgame", newGameFrame, true, false)
	rootPage.AddPage("login", loginFrame, true, false)
	rootPage.AddPage("browser", gameListFrame, true, false)
	rootPage.AddPage("gameview", gameFrame, true, false)
	rootPage.AddPage("themes", themeList, true, false)
	rootPage.AddPage("loading", loadingModal, false, false)
	rootPage.AddPage("error", errorModal, false, false)

	showError = func(msg string) {
		errorModal.SetText(msg)
		rootPage.ShowPage("error")
		app.SetFocus(errorModal)
	}
	buildMenu = func() {
		menu.Clear()
		menu.AddItem("New game", "Play against GNU Go on this computer", 'n', func() {
			rootPage.SwitchToPage("newgame")
			app.SetFocus(newGameForm)
		})
		if saved := savedSettings(); saved != nil {
			s := *saved
			menu.AddItem("Continue", fmt.Sprintf("Your %dx%d game against GNU Go (level %d)", s.Size, s.Size, s.Level), 'c', func() {
				startLocal(s, true)
			})
		}
		if api.OauthClientID != "" {
			menu.AddItem("Play online", "Your games on online-go.com", 'o', startOnline)
		}
		menu.AddItem("Themes", "How the board looks", 't', func() {
			rootPage.ShowPage("themes")
		})
		menu.AddItem("Quit", "", 'q', app.Stop)
	}
	menuFrame.AddText("termsuji: Go in the terminal", true, tview.AlignCenter, tcell.PaletteColor(3))
	startOnline = func() {
		if auth.Tokens.Refresh != "" && !api.AuthData.Authenticated {
			api.AuthenticateRefreshToken(auth.Tokens.Refresh)
		}
		if api.AuthData.Authenticated {
			storeAuthData(auth)
			refreshGames()
			rootPage.SwitchToPage("browser")
		} else {
			rootPage.SwitchToPage("login")
		}
	}
	showMenu()

	if err := app.SetRoot(rootPage, true).Run(); err != nil {
		panic(err)
	}
}

var (
	showError   func(string)
	buildMenu   func()
	startOnline func()
)

// showMenu goes back to the start menu, with Continue offered if a game is
// saved.
func showMenu() {
	buildMenu()
	rootPage.SwitchToPage("menu")
	app.SetFocus(menu)
}

// savedSettings is the saved local game's settings, or nil if there is none.
func savedSettings() *ui.LocalSettings {
	dir, err := dataDir()
	if err != nil {
		return nil
	}
	if _, err := os.Stat(filepath.Join(dir, ui.SaveName)); err != nil {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(dir, settingsName))
	if err != nil {
		return nil
	}
	var s ui.LocalSettings
	if json.Unmarshal(data, &s) != nil || s.Level < 1 {
		return nil
	}
	return &s
}

// startLocal starts GNU Go and a new local game, or the saved one.
func startLocal(s ui.LocalSettings, saved bool) {
	dir, err := dataDir()
	if err != nil {
		showError("Can't make the folder for saved games: " + err.Error())
		return
	}
	eng, err := engine.Start(engine.FindGnuGo(), s.Level, dir)
	if err != nil {
		showError("Can't start GNU Go: " + err.Error())
		return
	}
	if saved {
		_, err = eng.LoadSGF(ui.SaveName)
	} else {
		err = eng.NewGame(s.Size, s.Komi, s.Handicap)
		if err == nil {
			data, _ := json.Marshal(s)
			err = os.WriteFile(filepath.Join(dir, settingsName), data, 0o644)
		}
	}
	if err != nil {
		eng.Close()
		showError("Can't start the game: " + err.Error())
		return
	}
	rootPage.SwitchToPage("gameview")
	app.SetFocus(gameBoard.Box)
	gameBoard.StartLocal(eng, s, dir)
}

func refreshGames() {
	gameList.Clear()
	async(func() {
		lastRefresh = time.Now()
		gamesArray := api.GetGamesList()
		i := 0
		for _, game := range gamesArray.Games {
			if game.GameOver() {
				continue
			}
			gameID := game.ID
			gameList.AddItem(game.Name, game.Description(), rune('1'+i), func() {
				async(func() {
					gameBoard.Connect(gameID)
					rootPage.SwitchToPage("gameview")
				})
			})
			i++
		}
		gameListHint := fmt.Sprintf("r: refresh, t: themes, q: menu")
		gameListFrame.Clear().AddText(gameListHint, false, tview.AlignLeft, tcell.ColorDefault)
	})
}

// Helper function to show a loading screen while blocking functions are being called.
func async(f func()) {
	go func() {
		setLoading(true)
		app.Draw()
		f()
		setLoading(false)
		app.Draw()
	}()
}

// Stores authentication data from api package after successful authentication.
func storeAuthData(a *config.AuthData) {
	a.Username = api.AuthData.Player.Username
	a.UserID = api.AuthData.Player.ID
	a.Tokens.Refresh = api.AuthData.Oauth.RefreshToken
	a.Save()
}
