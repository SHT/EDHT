package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"github.com/brunocannavina/goahrs"
	"github.com/tajtiattila/vjoy"
	"log"
	"math"
	"os/exec"
	"strings"
	"time"
	"io"
)

// Controller Controller
type Controller struct {
	Offset     Vector3
	Ori        Vector3
	InnerOri   Vector3
	Gyro       Vector3
	Acc        Vector3
	TouchPad   Vector2
	Quaternion goahrs.Quaternion
}

// Vector3 asda
type Vector3 struct {
	X float64
	Y float64
	Z float64
}

// Vector2 sajnda
type Vector2 struct {
	X float64
	Y float64
}

// NormalizeAngle ased
func NormalizeAngle(angle float64, offset float64) float64 {
	return math.Mod(angle+180+offset, 360)
}

// OffsetAngle asd
func OffsetAngle(angle float64, offset float64) float64 {
	return math.Mod(angle+360+180-offset, 360)
}

// MultiplyAngle asd
func MultiplyAngle(angle float64, factor float64) float64 {
	angle -= 180
	angle *= factor
	angle += 180
	return angle
}

func TestAngle(angle float64) float64 {
	angle -= 180
	angle *= -7
	// if angle < -15 {
	//     angle = -180
	// } else if angle > 15 {
	//     angle = 180
	// } else {
	//     angle = 0
	// }
	angle += 180
	return angle
}

// MarshalAngle prepares the angle to be sent to vJoy
func MarshalAngle(angle float64) float32 {
	return float32(angle) / 360
}

// ParseAngle returns a float64 in the range -180 to 180
func ParseAngle(val int32) float64 {
	if val>>12 == 1 {
		// Sign bit is 1, so convert to positive integer
		val = val & 0x0FFF
		// val = val - 4096
	}
	if val > 2048 {
		val = val - 4096
	}
	return float64(val) / 4096 * 360
}

func ParseInt(val int32) float64 {
	return float64(val) / 256
}

// GetOrientation asd
func GetOrientation(data []byte) Vector3 {
	// x and y are reversed? dafuq google
	y := int32((uint16(data[1])&0x03)<<11 | (uint16(data[2])&0xFF)<<3 | (uint16(data[3])&0xE0)>>5)
	x := int32((uint16(data[3])&0x1F)<<8 | (uint16(data[4]) & 0xFF))
	z := int32((uint16(data[5])&0xFF)<<5 | (uint16(data[6])&0xF8)>>3)
	return Vector3{
		X: ParseAngle(x),
		Y: ParseAngle(y),
		Z: ParseAngle(z),
	}
}

// GetAccelerometer asd
func GetAccelerometer(data []byte) Vector3 {
	x := int32((uint16(data[6])&0x07)<<10 | (uint16(data[7])&0xFF)<<2 | (uint16(data[8])&0xC0)>>6)
	y := int32((uint16(data[8])&0x3F)<<7 | (uint16(data[9])&0xFE)>>1)
	z := int32((uint16(data[9])&0x01)<<12 | (uint16(data[10])&0xFF)<<4 | (uint16(data[11])&0xF0)>>4)
	return Vector3{
		X: ParseAngle(x),
		Y: ParseAngle(y),
		Z: ParseAngle(z),
	}
}

// GetGyroscope asd
func GetGyroscope(data []byte) Vector3 {
	x := int32((uint16(data[11])&0x0F)<<9 | (uint16(data[12])&0xFF)<<1 | (uint16(data[13])&0x80)>>7)
	y := int32((uint16(data[13])&0x7F)<<6 | (uint16(data[14])&0xFC)>>2)
	z := int32((uint16(data[14])&0x03)<<11 | (uint16(data[15])&0xFF)<<3 | (uint16(data[16])&0xE0)>>5)
	return Vector3{
		X: ParseAngle(x),
		Y: ParseAngle(y),
		Z: ParseAngle(z),
	}
}

func GetTouchPad(data []byte) Vector2 {
	x := int32((uint16(data[16])&0x1F)<<3 | (uint16(data[17])&0xE0)>>5)
	y := int32((uint16(data[17])&0x1F)<<3 | (uint16(data[18])&0xE0)>>5)
	return Vector2{
		X: ParseInt(x),
		Y: ParseInt(y),
	}
}

func GetResetButton(data []byte) bool {
	return (uint16(data[18]) & 0x2) > 0
}

func main() {
	ctrl := Controller{
		Quaternion: goahrs.Quaternion{},
	}

	avail := vjoy.Available()
	if !avail {
		log.Println("vJoyInterface.dll could not be found.")
		log.Println("Please add it to your PATH environment variable.")
		return
	}

	d, err := vjoy.Acquire(1)
	if err != nil {
		fmt.Println(err)
		return
	}
	d.Reset()

	axes := []*vjoy.Axis{
		d.Axis(vjoy.AxisX),
		d.Axis(vjoy.AxisY),
		d.Axis(vjoy.AxisZ),
		d.Axis(vjoy.AxisRX),
		d.Axis(vjoy.AxisRY),
		d.Axis(vjoy.AxisRZ),
		d.Axis(vjoy.Slider0),
		d.Axis(vjoy.Slider1),
	}

	cmd := exec.Command("BLEconsole.exe")
	// cmd := exec.Command("cmd", "echo", "hello")
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	w, _ := cmd.StdinPipe()
	o, _ := cmd.StdoutPipe()
	e, _ := cmd.StderrPipe()

	r := io.MultiReader(o, e)


	go func() {
		time.Sleep(1 * time.Second)
		w.Write([]byte("format Hex\n"))
		time.Sleep(2 * time.Second)
		w.Write([]byte("open Daydream controller\n"))
		// time.Sleep(2 * time.Second)
		// w.Write([]byte("subs 65109/Custom Characteristic: 00000001-1000-1000-8000-00805f9b34fb\n"))
	}()
	go func() {
		memory := make([]byte, 59)
		cnt := 0
		started := false
		s := bufio.NewScanner(r)
		s.Split(bufio.ScanBytes)
		for s.Scan() {
			// fmt.Print(string(s.Bytes()))
			if cnt < 59 {
				if !started {
					b := s.Bytes()
					if string(b) != "\n" && string(b) != "\r"  {
						memory[cnt] = b[0]
						cnt++
					} else {
						if cnt != 0 {
							str := string(memory)
							if strings.HasPrefix(str, "Current display format: Hex") {
								fmt.Println("Attempting to connect...")
								w.Write([]byte("open Daydream controller\n"))
								time.Sleep(1 * time.Second)
								w.Write([]byte("print %name\\n\n"))
							} else if strings.HasPrefix(str, "Device Daydream controller is unreachable.") {
								fmt.Println("Device not found. Searching...")
								w.Write([]byte("open Daydream controller\n"))
								time.Sleep(1 * time.Second)
								w.Write([]byte("print %name\\n\n"))
							} else if strings.HasPrefix(str, "Can't connect to Daydream controller.") {
								fmt.Println("Couldn't connect. Retrying...")
								w.Write([]byte("open Daydream controller\n"))
								time.Sleep(1 * time.Second)
								w.Write([]byte("print %name\\n\n"))
							} else if strings.HasPrefix(str, "Connecting to Daydream controller.") {
								fmt.Println("Connecting...")
							} else if strings.HasPrefix(str, "Daydream controller") {
								fmt.Println("Connected!")
								time.Sleep(1 * time.Second)
								w.Write([]byte("subs 65109/Custom Characteristic: 00000001-1000-1000-8000-00805f9b34fb\n"))
								started = true
								ctrl.Quaternion.Begin(60)
							} else {
								fmt.Printf("Unknown command: %s", string(memory))
							}
							memory = make([]byte, 59)
							cnt = 0

						}
					}
				} else {
					b := s.Bytes()
					memory[cnt] = b[0]
					cnt++
					if cnt == 59 {
						str := strings.Replace(string(memory), " ", "", -1)
						enc, err := hex.DecodeString(str)
						if err != nil {
							panic(err)
						}
						go Handle(enc, d, axes, &ctrl)
					}
				}
			}
			if cnt >= 59 {
				memory = make([]byte, 59)
				cnt = 0
			}
		}
		fmt.Println("Reached EOF. Exiting.")
		return
	}()
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

func Handle(data []byte, d *vjoy.Device, axes []*vjoy.Axis, ctrl *Controller) {

	reset := GetResetButton(data)
	ctrl.InnerOri = GetOrientation(data)
	ctrl.Acc = GetAccelerometer(data)
	ctrl.Gyro = GetGyroscope(data)
	ctrl.TouchPad = GetTouchPad(data)

	gyroX, gyroY, gyroZ := float64(ctrl.Gyro.X)/(65/(2*math.Pi)), float64(ctrl.Gyro.Y)/(65/(2*math.Pi)), float64(ctrl.Gyro.Z)/(65/(2*math.Pi))
	accX, accY, accZ := float64(ctrl.Acc.X), float64(ctrl.Acc.Y), float64(ctrl.Acc.Z)

	ctrl.Quaternion.UpdateIMU(gyroX, gyroY, gyroZ, accX, accY, accZ)

	roll := NormalizeAngle(ctrl.Quaternion.GetRoll(), 90)
	pitch := NormalizeAngle(ctrl.Quaternion.GetPitch(), 0)

	if reset {
		ctrl.Offset = Vector3{
			X: ctrl.InnerOri.X,
			Y: roll,
			Z: pitch,
		}
	}

	x := OffsetAngle(ctrl.InnerOri.X, ctrl.Offset.X)
	x = MultiplyAngle(x, -1)

	y := OffsetAngle(roll, ctrl.Offset.Y)
	y = MultiplyAngle(y, 1)

	axes[0].Setuf(MarshalAngle(x))
	axes[1].Setuf(MarshalAngle(y))

	_ = d.Update()

}
