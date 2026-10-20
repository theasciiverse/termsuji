package config

var DefaultConfig Config
var ClassicTheme Theme
var DefaultTheme Theme
var VaporwaveTheme Theme
var UnicodeTheme Theme
var CatdogTheme Theme
var HongokuTheme Theme

func init() {
	DefaultTheme = Theme{
		DrawStoneBackground:      true,
		DrawCursorBackground:     false,
		DrawLastPlayedBackground: false,
		FullWidthLetters:         false,
		Colors: ConfigColors{
			BoardColor:        220,
			BoardColorAlt:     221,
			BlackColor:        233,
			BlackColorAlt:     235,
			WhiteColor:        255,
			WhiteColorAlt:     254,
			CursorColorFG:     2,
			CursorColorBG:     4,
			LastPlayedColorBG: 2,
		},
		Symbols: ConfigSymbols{
			BlackStone:  ' ',
			WhiteStone:  ' ',
			BoardSquare: ' ',
			Cursor:      'X',
			LastPlayed:  '/',
		},
	}
	ClassicTheme = DefaultTheme
	ClassicTheme.Grid = true
	ClassicTheme.Colors.BoardColor = 179   //wood
	ClassicTheme.Colors.BoardColorAlt = 94 //lines
	ClassicTheme.Colors.BlackColor = 232
	ClassicTheme.Colors.WhiteColor = 231
	ClassicTheme.Colors.CursorColorBG = 33
	ClassicTheme.Colors.LastPlayedColorBG = 166
	ClassicTheme.Symbols.BlackStone = '●'
	ClassicTheme.Symbols.WhiteStone = '●'
	DefaultConfig = Config{
		Theme: ClassicTheme,
	}

	VaporwaveTheme = DefaultTheme
	VaporwaveTheme.Colors.BoardColor = 251
	VaporwaveTheme.Colors.BoardColorAlt = 252
	VaporwaveTheme.Colors.BlackColor = 164
	VaporwaveTheme.Colors.BlackColorAlt = 165
	VaporwaveTheme.Colors.WhiteColor = 87
	VaporwaveTheme.Colors.WhiteColorAlt = 51

	UnicodeTheme = DefaultTheme
	UnicodeTheme.DrawStoneBackground = false
	UnicodeTheme.DrawCursorBackground = true
	UnicodeTheme.DrawLastPlayedBackground = true
	UnicodeTheme.FullWidthLetters = true
	UnicodeTheme.Symbols.BlackStone = '⚫'
	UnicodeTheme.Symbols.WhiteStone = '⚪'
	UnicodeTheme.Symbols.BoardSquare = '➕'

	CatdogTheme = UnicodeTheme
	CatdogTheme.Symbols.BlackStone = '😺'
	CatdogTheme.Symbols.WhiteStone = '🐶'

	HongokuTheme = UnicodeTheme
	HongokuTheme.DrawCursorBackground = false
	HongokuTheme.DrawLastPlayedBackground = false
	HongokuTheme.Colors.BlackColor = 0
	HongokuTheme.Colors.WhiteColor = 0
	HongokuTheme.Colors.CursorColorFG = 0
	HongokuTheme.Symbols.BlackStone = '黒'
	HongokuTheme.Symbols.WhiteStone = '白'
	HongokuTheme.Symbols.BoardSquare = '空'
	HongokuTheme.Symbols.Cursor = '選'
	HongokuTheme.Symbols.LastPlayed = '前'
}
