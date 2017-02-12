package main

import (
    "os"
    "fmt"
    "log"
    "path"
    "bufio"
    "strconv"
    "strings"
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
  const maxImageBytes int = maxWidth * maxHeight * 8
  fmt.Println("KK", maxImageBytes)
  outputImageBaseName := destination + "/" + path.Base(inputFile) + ".GooglePhotosStorage"
  
  // Open the file to encode
  tmpInputFileReader, err := os.Open(inputFile)
  if err != nil {
    panic(err)
  }
  defer tmpInputFileReader.Close()
  inputFileReader := bufio.NewReader(tmpInputFileReader)
  buf := make([]byte, 8)

  var part int = 0
  var bytesRead int = 0

  for {

    log.Println("START PART", part)
    loopAgain := false

    // Create the image object
    outputImage := image.NewNRGBA64(image.Rect(0, 0, maxWidth, maxHeight))

    // Vars for loop
    x := 0
    maxX:= outputImage.Bounds().Max.X
    y := 0

    // Loop
    for {
      if bytesRead + 8 - (maxImageBytes * part) > maxImageBytes {
        fmt.Println("TOO MUCH", bytesRead, maxImageBytes)
        loopAgain = true
        // Set the reader at corret position
        break
      }
      count, _ := inputFileReader.Read(buf)
      bytesRead += count
      if count == 0{
        fmt.Println("THE EOF, ending", bytesRead)
        break
      }
      // Calculate the pixel color
      var sliceBuf = make([]byte, 0, 8)
      sliceBuf = buf
      // For all not read, complete with 0
      for i := count; i < 8; i ++ {
        sliceBuf[i] = 0
      }
      pixelColor := color.NRGBA64{
        binary.BigEndian.Uint16(buf[0:2]),
        binary.BigEndian.Uint16(buf[2:4]),
        binary.BigEndian.Uint16(buf[4:6]),
        binary.BigEndian.Uint16(buf[6:8]),
      }
      // Set it
      outputImage.Set(x, y, pixelColor)
      // Update the next pixel position
      x++
      if (x >= maxX) {
        x = 0
        y++
      }
    }
    part += 1

    // Choose the name
    var outputImageName string
    if (!loopAgain && part == 1) {
      outputImageName = outputImageBaseName + ".png"
    } else {
      outputImageName = outputImageBaseName + ".part" + strconv.Itoa(part) + ".png"
    }

    // The file (part) destination
    log.Println("Writing to", outputImageName)
    outputFile, err := os.OpenFile(outputImageName,
      os.O_WRONLY|os.O_TRUNC|os.O_CREATE,0600,)
    if err != nil {
        panic(err)
    }
    // Save the part !
    png.Encode(outputFile, outputImage)
    outputFile.Close()

    if (!loopAgain) {
      break
    }

  } // While Part(s)
  fmt.Println("TOTAL", bytesRead)

}

func decodeFile(files []string, destination string) {

  name := path.Base(files[0])
  baseName := name[0:strings.LastIndex(name, ".GooglePhotosStorage")]
  outputFileName := destination + "/" + baseName
  f, err := os.OpenFile(outputFileName, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0600)
  if err != nil {
      panic(err)
  }
  f.Write([]byte{})
  f.Close()

  lastIndexFile := len(files) - 1

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
      if value == 0 && lastIndexFile == indexFile && isSliceAllZero(inputBuffer[index+1:index+10])  {
        break
      } else {
        inputBufferCleaned = append(inputBufferCleaned, value)
      }
    }

    f, err := os.OpenFile(
        outputFileName,
        os.O_WRONLY| os.O_APPEND,
        0600,
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
