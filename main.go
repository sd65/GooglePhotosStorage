package main

import (
    "image"
    "image/color"
    "image/png"
    "image/draw"
    "os"
    "io/ioutil"
    "encoding/binary"
    "fmt"
)


func main() {

    // Args parsing
    action := os.Args[1]
    file := os.Args[2]

    if action == "-e" {

      // Create the Image
      img := image.NewNRGBA64(image.Rect(0, 0, 4614, 3464))

      // The file destination
      imgFile, err := os.Create("draw.png")
      if err != nil {
          panic(err)
      }
      defer imgFile.Close()

      // The file to encode
      inputBuffer, err := ioutil.ReadFile(file)
      if err != nil {
        panic(err)
      }
      fmt.Println(inputBuffer)

      // Utils
      x := 0
      y := 0
      returnNextPixel := func() {
        x++
        if (x >= img.Bounds().Dx()) {
          x = 0
          y++
        }
      }

      // Loop
      for i := 0; i < len(inputBuffer); i += 8 {
        fmt.Println("Making a Pixel")
        takeDupleAtPosition := func(n int) uint16{
          if n >= len(inputBuffer) {
            fmt.Println("Out of range")
            return uint16(0)
          }
          var b1 uint8 = inputBuffer[n]
          var b2 uint8
          if n+1 >= len(inputBuffer) {
            b2 = 0
          } else {
            b2 = inputBuffer[n+1]
          }
          tmpSlice := []byte{b1, b2}
      	  return binary.BigEndian.Uint16(tmpSlice)
        }
        n1 := takeDupleAtPosition(i)
        n2 := takeDupleAtPosition(i + 2)
        n3 := takeDupleAtPosition(i + 4)
        n4 := takeDupleAtPosition(i + 6)
        fmt.Printf("Values : %d %d %d %d \n", n1, n2, n3, n4)
        color := color.NRGBA64{n1, n2, n3, n4}
        img.Set(x, y, color)
        returnNextPixel()
      }

      png.Encode(imgFile, img)

    } else { // Decode

      inputFile, err := os.Open(file)
      if err != nil {
          panic(err)
      }
      defer inputFile.Close()

      img, _, err := image.Decode(inputFile)
      if err != nil {
          panic(err)
      }


      rect := img.Bounds()
      myDecodedPicture := image.NewNRGBA64(rect)
      draw.Draw(myDecodedPicture, rect, img, rect.Min, draw.Src)
      pixs := myDecodedPicture.Pix

      allZero := func(slice []uint8) bool {
        for _, value := range slice {
          if value != 0 {
            return false
          }
        }
        return true
      }
      
      fb := make([]byte, 0)
      //fb := make([]byte, len(pixs))
      for index, value := range pixs {
        if value == 0 && allZero(pixs[index+1:index+500])  {
          break
        } else {
          fb = append(fb, value)
        }
      }

      fmt.Println(fb[:600])
      ioutil.WriteFile("result", fb, 0644)


    }
}
