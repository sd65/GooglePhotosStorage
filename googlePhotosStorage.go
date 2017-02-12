package main

import (
    "os"
    "fmt"
    "log"
    "path"
    "strconv"
    "io/ioutil"
    "encoding/binary"
    "image"
    "image/color"
    "image/png"
    "image/draw"

    "github.com/alexflint/go-arg"
)

type args struct {
  Encode string `arg:"-e,help: encode file to PNG"`
  Decode []string `arg:"-d,help: decode file from PNG"`
  Destination string
}

func encodeFile(inputFile string, destination string) {

  // Max size for Google Photos
  const maxWidth int = 4614
  const maxHeight int = 3464
  const maxImageSize int = maxWidth * maxWidth * 8
  
  // Open the file to encode
  inputBufferArray, err := ioutil.ReadFile(inputFile)
  if err != nil {
    panic(err)
  }
  inputBuffer := inputBufferArray[:]

  // Utils
  valueOrZeroOnSlice := func(slice *[]uint8, index int) uint8 {
    if index >= len(*slice) {
        fmt.Println("Out of range")
        return 0
    }
    return (*slice)[index]
  }

  var part int64 = 0
  var loopAgain bool = true
  var basePart int = 0

  for loopAgain == true {
    fmt.Println("START PART")

    loopAgain = false // No do while...

    // Create the image object
    outputImage := image.NewNRGBA64(image.Rect(0, 0, maxWidth, maxHeight))

    // The file (part) destination
    outputFile, err := os.Create(destination + "/" + path.Base(inputFile) + ".GPS.part" + strconv.FormatInt(part, 10) + ".png")
    if err != nil {
        panic(err)
    }
    defer outputFile.Close()

    // Loop
    var x int = 0
    var y int = 0
    for i := basePart; i < len(inputBuffer); i += 8 { // For each 8 bytes == 64 bits
      if i - basePart > maxImageSize {
         fmt.Println("TOO MUCH")
         fmt.Println(basePart)
         fmt.Println(i)
         fmt.Println(maxImageSize)
        // Too large !
        loopAgain = true
        basePart = i
        part += 1
        break
      }
      // Calculate the pixel color
      var pixelValues = make([]uint16, 0, 4)
      for y := 0; y < 8; y += 2 {
        b1 := valueOrZeroOnSlice(&inputBuffer, i + y) 
        b2 := valueOrZeroOnSlice(&inputBuffer, i + y + 1) 
        tmpSlice := []byte{b1, b2}
        pixelValues = append(pixelValues, binary.BigEndian.Uint16(tmpSlice))
      }
      pixelColor := color.NRGBA64{pixelValues[0], pixelValues[1], pixelValues[2], pixelValues[3]}
      // Set it
      outputImage.Set(x, y, pixelColor)
      // Update the next pixel position
      x++
      if (x >= outputImage.Bounds().Max.X) {
        x = 0
        y++
      }
    }

    // Save the part !
    png.Encode(outputFile, outputImage)

  } // While Part(s)
}

func decodeFile(files []string, destination string) {

  outputFileName := destination + "/OUT"
  f, err := os.OpenFile(outputFileName, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0777)
  if err != nil {
      panic(err)
  }
  f.Write([]byte{})
  f.Close()

  for indexFile, file := range files {
    fmt.Println("NEW FILE")

    inputFile, err := os.Open(file)
    if err != nil {
        panic(err)
    }
    defer inputFile.Close()

    inputImage, _, err := image.Decode(inputFile)
    if err != nil {
        log.Fatal("Error while decoding the file. Is your file an PNG image created by this tool?")
    }

    inputImageBounds := inputImage.Bounds()
    inputImageD := image.NewNRGBA64(inputImageBounds)
    draw.Draw(inputImageD, inputImageBounds, inputImage, inputImageBounds.Min, draw.Src)
    inputBuffer := inputImageD.Pix

    isSliceAllZero := func(slice []uint8) bool {
      for _, value := range slice {
        if value != 0 {
          return false
        }
      }
      return true
    }
    
    inputBufferCleaned := make([]byte, 0, len(inputBuffer))
    for index, value := range inputBuffer {
      if value == 0 && isSliceAllZero(inputBuffer[index+1:index+300])  {
        break
      } else {
        inputBufferCleaned = append(inputBufferCleaned, value)
      }
    }


    f, err := os.OpenFile(
        outputFileName + strconv.Itoa(indexFile),
        os.O_WRONLY|os.O_APPEND | os.O_CREATE,
        777,
    )
    if err != nil {
        log.Fatal(err)
    }
    bytesWritten, err := f.Write(inputBufferCleaned)
    if err != nil {
        log.Fatal(err)
    }
    f.Close()
    fmt.Println(bytesWritten)
    //ioutil.WriteFile(outputFileName, inputBufferCleaned, os.ModeAppend | os.O_WRONLY)

  } // End of loop encoded files
}

func (args) Version() string {
    return "GooglePhotosStorage 1.0"
}

func (args) Description() string {
    return "this program does this and that"
}

func pathExist(path string, mustBeDir bool) bool {
  stat, err := os.Stat(path)
  if err != nil {
    return false
  }
  if mustBeDir && ! stat.IsDir() {
    return false
  }     
  return true
}

func main() {

    // Args parsing
    var args args
    p := arg.MustParse(&args)


    // Checking
    if args.Encode == "" && len(args.Decode) == 0 || args.Encode != "" && len(args.Decode) != 0 {
    p.Fail("you must provide one of --encode and --decode")
    }

    // Switch action + additional checks
    if args.Encode != "" {
      if ! pathExist(args.Encode, false) {
        p.Fail("you must specify a EXISTING file to encode")
      }
      if args.Destination == "" {
        args.Destination = "encoded"
      }
      if ! pathExist(args.Destination, true) {
        p.Fail("you must specify a EXISTING dir for the encoded file(s)")
      }
      encodeFile(args.Encode, args.Destination)
    } else if len(args.Decode) != 0 {
      for _, file := range args.Decode {
        if ! pathExist(file, false) {
          p.Fail("you must specify a EXISTING file(s) to decode")
        }
      }
      if args.Destination == "" {
        args.Destination = "decoded"
      }
      if ! pathExist(args.Destination, false) {
        p.Fail("you must specify a EXISTING dir for the decoded file")
      }
      decodeFile(args.Decode, args.Destination)
    }
}
