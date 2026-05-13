package hotkey

import (
	"fmt"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// Modificadores soportados
const (
	ModShift = xproto.ModMaskShift
	ModCtrl  = xproto.ModMaskControl
	ModAlt   = xproto.ModMask1
)

type binding struct {
	modifiers uint16
	keycode   xproto.Keycode
	callback  func()
}

type XGBListener struct {
	conn     *xgb.Conn
	root     xproto.Window
	bindings []binding
	done     chan struct{}
}

func NewXGBListener() (*XGBListener, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to X11: %w", err)
	}

	setup := xproto.Setup(conn)
	root := setup.DefaultScreen(conn).Root

	return &XGBListener{
		conn: conn,
		root: root,
		done: make(chan struct{}),
	}, nil
}

// Register acepta un string como "ctrl+shift+v" y registra el callback.
func (l *XGBListener) Register(keys string, callback func()) error {
	mods, keycode, err := l.parseKeys(keys)
	if err != nil {
		return err
	}

	fmt.Printf("[hotkey] registering %q → modifiers=0x%x keycode=%d\n", keys, mods, keycode)

	cookie := xproto.GrabKeyChecked(
		l.conn, true, l.root,
		mods, keycode,
		xproto.GrabModeAsync, xproto.GrabModeAsync,
	)
	if err := cookie.Check(); err != nil {
		return fmt.Errorf("failed to register hotkey %q: %w", keys, err)
	}

	fmt.Printf("[hotkey] registered successfully\n")
	l.bindings = append(l.bindings, binding{mods, keycode, callback})
	return nil
}

func (l *XGBListener) Listen() error {
	go func() {
		defer close(l.done)
		for {
			ev, err := l.conn.WaitForEvent()
			if err != nil {
				return
			}
			if kp, ok := ev.(xproto.KeyPressEvent); ok {
				activeMods := kp.State & ^uint16(xproto.ModMaskLock|xproto.ModMask2)
				fmt.Printf("[hotkey] key press: keycode=%d modifiers=0x%x\n", kp.Detail, activeMods)
				for _, b := range l.bindings {
					if kp.Detail == b.keycode && activeMods == b.modifiers {
						fmt.Printf("[hotkey] match! firing callback\n")
						go b.callback()
					}
				}
			}
		}
	}()
	return nil
}

func (l *XGBListener) Stop() error {
	l.conn.Close()
	<-l.done
	return nil
}

func (l *XGBListener) parseKeys(keys string) (uint16, xproto.Keycode, error) {
	keySyms, err := xproto.GetKeyboardMapping(
		l.conn,
		xproto.Setup(l.conn).MinKeycode,
		uint8(xproto.Setup(l.conn).MaxKeycode-xproto.Setup(l.conn).MinKeycode+1),
	).Reply()
	if err != nil {
		return 0, 0, err
	}

	symMap := map[string]uint32{
		"a": 0x61, "b": 0x62, "c": 0x63, "d": 0x64, "e": 0x65,
		"f": 0x66, "g": 0x67, "h": 0x68, "i": 0x69, "j": 0x6a,
		"k": 0x6b, "l": 0x6c, "m": 0x6d, "n": 0x6e, "o": 0x6f,
		"p": 0x70, "q": 0x71, "r": 0x72, "s": 0x73, "t": 0x74,
		"u": 0x75, "v": 0x76, "w": 0x77, "x": 0x78, "y": 0x79,
		"z": 0x7a,
	}

	var mods uint16
	var targetSym uint32

	parseToken := func(token string) error {
		switch token {
		case "ctrl":
			mods |= ModCtrl
		case "shift":
			mods |= ModShift
		case "alt":
			mods |= ModAlt
		default:
			sym, ok := symMap[token]
			if !ok {
				return fmt.Errorf("unknown key: %q", token)
			}
			targetSym = sym
		}
		return nil
	}

	start := 0
	for i := 0; i <= len(keys); i++ {
		if i == len(keys) || keys[i] == '+' {
			token := keys[start:i]
			if token != "" {
				if err := parseToken(token); err != nil {
					return 0, 0, err
				}
			}
			start = i + 1
		}
	}

	if targetSym == 0 {
		return 0, 0, fmt.Errorf("no main key specified in %q", keys)
	}

	perSym := int(keySyms.KeysymsPerKeycode)
	minKeycode := int(xproto.Setup(l.conn).MinKeycode)

	for i, sym := range keySyms.Keysyms {
		if uint32(sym) == targetSym {
			keycode := xproto.Keycode(minKeycode + i/perSym)
			return mods, keycode, nil
		}
	}

	return 0, 0, fmt.Errorf("keycode not found for symbol 0x%x", targetSym)
}
