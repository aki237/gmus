package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"

	"github.com/dhowden/tag"
	"github.com/nfnt/resize"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	colors "gopkg.in/go-playground/colors.v1"
)

var DIMEN = 500

type Config struct {
	DynamicBG bool
	BGColor   string
	FGColor   string
	Font      string
	FontSize  int
}

func main() {
	c := &Config{}
	flag.StringVar(&c.BGColor, "bg", "#000000", "background color to paint the application")
	flag.IntVar(&c.FontSize, "s", 14, "size of the rendered font")
	flag.StringVar(&c.FGColor, "fg", "#FFFFFF", "background color to paint the application")
	flag.StringVar(&c.Font, "fnt", "", "choose font")
	flag.BoolVar(&c.DynamicBG, "dyn", false, "choose the back ground color adaptively from the album art")

	flag.Parse()

	if c.Font == "" {
		return
	}

	bg, err := colors.ParseHEX(c.BGColor)
	if err != nil {
		fmt.Println(err, "Falling back to black background")
		bg, _ = colors.ParseHEX("#000000")
	}

	fg, err := colors.ParseHEX(c.FGColor)
	if err != nil {
		fmt.Println(err, "Falling back to white foreground")
		bg, _ = colors.ParseHEX("#000000")
	}

	ss, err := newCmusSocket()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ss.conn.Close()

	err = sdl.Init(sdl.INIT_VIDEO)
	if err != nil {
		fmt.Println(err)
		return
	}

	window, err := sdl.CreateWindow("gmus", 50, 50, int32(DIMEN), int32(DIMEN), sdl.WINDOW_SHOWN)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer window.Destroy()

	if err := ttf.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize TTF: %s\n", err)
		return
	}

	font, err := ttf.OpenFont(c.Font, c.FontSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open font: %s\n", err)
		return
	}
	defer font.Close()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create renderer: %s\n", err)
		return
	}
	defer renderer.Destroy()
	running := true

	var img image.Image

	dragging := false

	prevFile := ""
	sw := 1
	scalex := float32(1.0)
	scaley := float32(1.0)
	ww, wh := window.GetSize()
	isSeekAnim := false
	seekanim := 0
	alpha := 255
	da := -1
	for running {
		s, err := ss.GetStatus()
		if err != nil {
			fmt.Println(err)
			break
		}

		if !s.Playing {

			if alpha >= 255 {
				da = -1
			}
			if alpha <= 100 {
				da = 1
			}
			alpha += 5 * da
		}

		renderer.SetScale(scalex, scaley)
		renderer.Clear()

		if s.File != prevFile {
			fl, err := os.Open(s.File)
			if err != nil {
				fmt.Println(err)
				continue
			}

			m, err := tag.ReadFrom(fl)
			if err == nil {
				if prevFile != s.File {
					pic := m.Picture()
					if pic != nil {
						img, _, err = image.Decode(bytes.NewBuffer(pic.Data))
						if err != nil {
							fmt.Println(err)
						} else {
							img = resize.Resize(200, 200, img, resize.Lanczos3)
						}
					}
				}
			}
			fl.Close()
		}

		if c.DynamicBG && img != nil {
			clr := imageBaseColor(img)
			renderer.SetDrawColor(clr.R, clr.G, clr.B, 255)
			if clr.IsDark() {
				fg, _ = colors.ParseHEX("#DDDDDD")
			} else {
				fg, _ = colors.ParseHEX("#222222")
			}
		} else {
			renderer.SetDrawColor(bg.ToRGB().R, bg.ToRGB().G, bg.ToRGB().B, 255)
		}
		renderer.FillRect(&sdl.Rect{X: 0, Y: 0, W: int32(DIMEN), H: int32(DIMEN)})

		// album art
		if img != nil {
			blitAtCircle(renderer, img, int32(DIMEN/2)-100, 100, uint8(alpha))
		}

		txt := ""
		if len(s.Title) > 16 {
			txt += s.Title[:16] + "...  -  "
		} else {
			txt += s.Title + "  -  "
		}
		if len(s.Artist) > 16 {
			txt += s.Artist[:16] + "..."
		} else {
			txt += s.Artist
		}
		rgb := fg.ToRGB()
		DrawText(renderer, txt, font, rgb, 350)
		DrawText(renderer, s.Album, font, rgb, 380)

		e := float64(DIMEN) * (float64(s.Position) / float64(s.Duration))
		renderer.SetDrawColor(fg.ToRGB().R, fg.ToRGB().G, fg.ToRGB().G, uint8(255))
		// Seek
		renderer.FillRect(&sdl.Rect{X: 0, Y: 420, W: int32(e), H: int32(1)})
		renderer.FillRect(&sdl.Rect{X: int32(e), Y: 420 - int32(sw/2), W: 2, H: int32(sw)})

		// Seek Timing
		if sw == 20 {
			DrawTextAt(renderer, fmt.Sprintf("%02d:%02d", (s.Position-(s.Position%60))/60, s.Position%60), font, rgb, int32(e), 395)
		}

		// Previous
		renderer.DrawLine(int32(DIMEN/2)-50, int32(DIMEN-50), int32(DIMEN/2)-65, int32(DIMEN-40)) // /
		renderer.DrawLine(int32(DIMEN/2)-50, int32(DIMEN-50), int32(DIMEN/2)-50, int32(DIMEN-30)) //  |
		renderer.DrawLine(int32(DIMEN/2)-50, int32(DIMEN-30), int32(DIMEN/2)-65, int32(DIMEN-40)) // \
		renderer.DrawRect(&sdl.Rect{X: int32(DIMEN/2) - 70, Y: int32(DIMEN - 50), W: 5, H: 20})   //|

		// Next
		renderer.DrawLine(int32(DIMEN/2)+50, int32(DIMEN-50), int32(DIMEN/2)+50, int32(DIMEN-30))
		renderer.DrawLine(int32(DIMEN/2)+50, int32(DIMEN-50), int32(DIMEN/2)+65, int32(DIMEN-40))
		renderer.DrawLine(int32(DIMEN/2)+50, int32(DIMEN-30), int32(DIMEN/2)+65, int32(DIMEN-40))
		renderer.DrawRect(&sdl.Rect{X: int32(DIMEN/2) + 65, Y: int32(DIMEN - 50), W: 5, H: 20}) //|

		// Pause/Play
		if s.Playing {
			renderer.DrawRect(&sdl.Rect{X: int32(DIMEN/2) - 10, Y: int32(DIMEN - 50), W: 8, H: 20}) //|
			renderer.DrawRect(&sdl.Rect{X: int32(DIMEN/2) + 2, Y: int32(DIMEN - 50), W: 8, H: 20})  // |
		} else {
			renderer.DrawLine(int32(DIMEN/2)-10, int32(DIMEN-50), int32(DIMEN/2)+10, int32(DIMEN-40)) //  \
			renderer.DrawLine(int32(DIMEN/2)-10, int32(DIMEN-50), int32(DIMEN/2)-10, int32(DIMEN-30)) // |
			renderer.DrawLine(int32(DIMEN/2)+10, int32(DIMEN-40), int32(DIMEN/2)-10, int32(DIMEN-30)) //  /
		}

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.WindowEvent:
				if t.Event != sdl.WINDOWEVENT_RESIZED {
					break
				}
				scalex = float32(t.Data1) / float32(ww)
				scaley = float32(t.Data2) / float32(wh)
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseWheelEvent:
				if t.Y > 0 {
					ss.Seek(s.Position + 5)
				}
				if t.Y < 0 {
					ss.Seek(s.Position - 5)
				}
			case *sdl.KeyboardEvent:
				if t.State == sdl.PRESSED {
					switch t.Keysym.Sym {
					case sdl.K_RIGHT:
						ss.Seek(s.Position + 1)
					case sdl.K_LEFT:
						if s.Position-1 < 0 {
							ss.Seek(0)
						} else {
							ss.Seek(s.Position - 1)
						}
					case sdl.K_HOME:
						ss.Seek(0)
					case sdl.K_END:
						ss.Seek(s.Duration)
					case sdl.K_SPACE:
						ss.TogglePausePlay()
					case sdl.K_n:
						ss.Next()
					case sdl.K_p:
						ss.Prev()
					case sdl.K_PLUS, sdl.K_EQUALS:
						ss.VolUp()
					case sdl.K_MINUS, sdl.K_UNDERSCORE:
						ss.VolDown()
					}
				}
			case *sdl.MouseMotionEvent:
				if dragging {
					ss.Seek(int(float64(t.X) * float64(s.Duration) / float64(DIMEN)))
				}
				if !dragging {
					if t.Y >= 418 && t.Y <= 423 {
						isSeekAnim = true
					} else {
						isSeekAnim = false
					}
				}
			case *sdl.MouseButtonEvent:
				if t.Button == sdl.BUTTON_LEFT && t.State == sdl.RELEASED {

					// Seek press
					if t.Y >= 418 && t.Y <= 423 {
						ss.Seek(int(float64(t.X) * float64(s.Duration) / float64(DIMEN)))
					}

					// Control Sector
					if t.Y >= int32(DIMEN-50) && t.Y <= int32(DIMEN-30) {
						// Previous button
						if t.X <= int32(DIMEN/2)-50 && t.X >= int32(DIMEN/2)-65 {
							ss.Prev()
						}
						// Next Button
						if t.X >= int32(DIMEN/2)+50 && t.X <= int32(DIMEN/2)+65 {
							ss.Next()
						}
						// Pause button
						if t.X >= int32(DIMEN/2)-10 && t.X <= int32(DIMEN/2)+10 {
							ss.TogglePausePlay()
						}
					}
				}
				if t.Button == sdl.BUTTON_LEFT && t.State == sdl.PRESSED && t.Y >= 418 && t.Y <= 423 {
					dragging = true
				} else {
					dragging = false
				}
			}
		}
		if isSeekAnim {
			seekanim += 4
			if seekanim >= 20 {
				seekanim = 20
			}
		} else {
			seekanim -= 4
			if seekanim <= 0 {
				seekanim = 0
			}
		}
		sw = seekanim
		prevFile = s.File
		renderer.Present()
		sdl.Delay(30)
	}
}

func dist(x1, y1, x2, y2 int32) float64 {
	return math.Sqrt(float64(math.Pow(float64(x2-x1), 2) + math.Pow(float64(y2-y1), 2)))
}

func blitAtCircle(renderer *sdl.Renderer, img image.Image, X, Y int32, alpha uint8) {
	renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
	cx := int32(img.Bounds().Dx() / 2)
	cy := int32(img.Bounds().Dy() / 2)
	r := img.Bounds().Dy() / 2
	for y := 0; y < img.Bounds().Max.Y; y++ {
		for x := 0; x < img.Bounds().Max.X; x++ {
			if dist(cx, cy, int32(x), int32(y)) < float64(r) {
				r, g, b, _ := img.At(x, y).RGBA()
				renderer.SetDrawColor(uint8(r&r>>8), uint8(g&g>>8), uint8(b&b>>8), alpha)
				renderer.DrawPoint(int32(X)+int32(x), int32(Y)+int32(y))
			}
		}
	}
}

func blitAt(renderer *sdl.Renderer, img image.Image, X, Y int32, alpha uint8) {
	for y := 0; y < img.Bounds().Max.Y; y++ {
		for x := 0; x < img.Bounds().Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			renderer.SetDrawColor(uint8(r&r>>8), uint8(g&g>>8), uint8(b&b>>8), alpha)
			renderer.DrawPoint(int32(X)+int32(x), int32(Y)+int32(y))
		}
	}
}

func DrawText(renderer *sdl.Renderer, text string, font *ttf.Font, rgb *colors.RGBColor, y int32) {
	renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
	solid, err := font.RenderUTF8Blended(text, sdl.Color{R: rgb.R, G: rgb.G, B: rgb.B, A: 255})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render text: %s\n", err)
		return
	}
	tex, _ := renderer.CreateTextureFromSurface(solid)
	renderer.Copy(tex, nil, &sdl.Rect{X: (int32(DIMEN) - solid.W) / 2, Y: y, W: solid.W, H: solid.H})
	solid.Free()
	tex.Destroy()
}

func DrawTextAt(renderer *sdl.Renderer, text string, font *ttf.Font, rgb *colors.RGBColor, x, y int32) {
	renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
	solid, err := font.RenderUTF8Blended(text, sdl.Color{R: rgb.R, G: rgb.G, B: rgb.B, A: 255})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to render text: %s\n", err)
		return
	}
	tex, _ := renderer.CreateTextureFromSurface(solid)
	drx := x - int32(solid.W/2)
	if x+int32(solid.W/2) > int32(DIMEN) {
		drx = int32(DIMEN) - int32(solid.W)
	}
	if x-int32(solid.W/2) < 0 {
		drx = 0
	}
	renderer.Copy(tex, nil, &sdl.Rect{X: drx, Y: y, W: solid.W, H: solid.H})
	solid.Free()
	tex.Destroy()
}
