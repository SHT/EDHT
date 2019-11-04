package main

import (
	"bufio"
	"fmt"
	"github.com/brunocannavina/goahrs"
	"github.com/tajtiattila/vjoy"
	"io"
	"log"
	"math"
	"net"
	"time"
)

// Listen sets up a listener and returns the connection and the listener
func Listen() (net.Conn, net.Listener, error) {
	// Format address string
	address := fmt.Sprintf(":%d", 4198)
	// Establish a listener
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, err
	}
	// Accept incoming connections
	conn, err := listener.Accept()
	if err != nil {
		return nil, listener, err
	}
	fmt.Printf("Incoming connection from %s\n", conn.RemoteAddr())
	return conn, listener, nil
}

// Receive reads the raw bytes from the socket, stores them in a buffer and
// then returns said buffer
func Receive(conn net.Conn, reader *bufio.Reader) ([]byte, error) {
	// Allocate buffer to store the incoming data
	// One more byte for the mode char
	buffer := make([]uint8, 20)
	// Create and store a socket reader on the struct
	// if it isn't already created

	// Read the data
	_, err := io.ReadFull(reader, buffer)
	if err != nil {
		return nil, err
	}

	// Return the data
	return buffer, nil
}

// DisconnectListener disconnects the listener when an error occurs while
// receiving data
func DisconnectListener(listener net.Listener) error {
	err := listener.Close()
	return err
}

// Controller Controller
type Controller struct {
	Offset     Vector3
	Ori        Vector3
	InnerOri   Vector3
	Gyro       Vector3
	Acc        Vector3
	Touch      Vector2
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
	X int
	Y int
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

func getResetButton(data []byte) bool {
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
	// Create a new listener on the specified port
	for {
		// Establish connection to the local socket
		conn, listener, err := Listen()
		if err != nil {
			time.Sleep(1 * time.Second)
			// Attempt to connect every second
			continue
		}
		reader := bufio.NewReader(conn)
		ctrl.Quaternion.Begin(60)

		// Stop channel is used to signal any running goroutine to stop
		// rendering and return
		// Initialize mode character
		// Receive data indefinitely
		for {
			data, err := Receive(conn, reader)
			if err != nil {
				// Disconnect the listener
				err := DisconnectListener(listener)
				if err != nil {
					log.Fatalf("Connection could not be closed: %s.\n", err)
				}
				// Connection lost, break the loop and try to reconnect
				break
			}
			// Split mode from data
			// Render the leds based on the given mode and data
			go Handle(data, d, axes, &ctrl)
		}
		// Try to reconnect every second
		time.Sleep(1 * time.Second)
	}
}

func Handle(data []byte, d *vjoy.Device, axes []*vjoy.Axis, ctrl *Controller) {

	reset := getResetButton(data)
	ctrl.InnerOri = GetOrientation(data)
	ctrl.Acc = GetAccelerometer(data)
	ctrl.Gyro = GetGyroscope(data)

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
	x = MultiplyAngle(x, 2)
	y := OffsetAngle(roll, ctrl.Offset.Y)
	y = MultiplyAngle(y, 2)

	axes[0].Setuf(MarshalAngle(x))

	axes[1].Setuf(MarshalAngle(y))

	_ = d.Update()

}
