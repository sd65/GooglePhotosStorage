package main

import (
    "os"
    "log"
    "fmt"
    "path"
    "bytes"
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

// The max bytes Google Photos accepts to upload
const maxGooglePhotoImageSize int = 70 * 1024 * 1024
// The end marker append to the picture files
var endMarker = []byte("<<••—{Th¡sIs±The†For" + "GooglePhotosStorage…}—••>>") // 8 bytes

// For args parsing
type args struct {
  Encode string `arg:"-e,help: encode file to PNG"`
  Decode []string `arg:"-d,help: decode file from PNG"`
  Destination string
}

// For help and version
func (args) Version() string {
    return "GooglePhotosStorage 1.0"
}

func (args) Description() string {
    return "this program does this and that"
}

// Funcs

func encodeFile(inputFile string, destination string) {

  // Make a square
  const size int = 3030

  // Get the name
  outputImageBaseName := destination + "/" + path.Base(inputFile) + ".GooglePhotosStorage"
  
  // Open the file to encode
  fmt.Printf("Opening %s to encode it...\n", inputFile)
  tmpInputFileReader, err := os.Open(inputFile)
  if err != nil {
    panic(err)
  }
  defer tmpInputFileReader.Close()
  inputFileReader := bufio.NewReader(tmpInputFileReader)

  // Prepare the buffer
  buf := make([]byte, 8)

  // Useful vars
  var part int = 0 // The part number if multiple image are created
  var writeIndexMarker int = 0 // Position of written end marker + flag
  var bytesRead int = 0 // A total of bytes read

  for { // Loop for each picture file created

    loopAgain := false // By default, one Picture is enough

    // Create the image object
    outputImage := image.NewNRGBA64(image.Rect(0, 0, size, size))

    // Vars for loop
    x := 0
    maxX:= outputImage.Bounds().Max.X
    y := 0

    // Loop on buffer read
    for {
      // If inputFile too large, make another image
      if bytesRead + 8 - (maxGooglePhotoImageSize * part) > maxGooglePhotoImageSize {
        fmt.Println("This file is too large, we will create another image.")
        loopAgain = true
        break
      }
      // Read the file
      count, _ := inputFileReader.Read(buf)
      bytesRead += count
      // If EOF reached, complete with end marker
      for i := count; i < 8; i ++ {
        buf[i] = endMarker[writeIndexMarker]
        writeIndexMarker++
        if (writeIndexMarker == 64) {
          writeIndexMarker = -1
          break
        }
      }
      // Create the pixel
      pixelColor := color.NRGBA64{
        binary.BigEndian.Uint16(buf[0:2]),
        binary.BigEndian.Uint16(buf[2:4]),
        binary.BigEndian.Uint16(buf[4:6]),
        binary.BigEndian.Uint16(buf[6:8]),
      }
      // Set it
      outputImage.Set(x, y, pixelColor)
      // Exit if end marker is written
      if writeIndexMarker == -1 {
        break
      }
      // Update the next pixel position
      x++
      if (x >= maxX) {
        x = 0
        y++
      }
    }

    // Choose the name of the picture
    part += 1
    var outputImageName string
    if (!loopAgain && part == 1) {
      outputImageName = outputImageBaseName + ".png"
    } else {
      outputImageName = outputImageBaseName + ".part" + strconv.Itoa(part) + ".png"
    }

    // Open the file destination
    outputFile, err := os.OpenFile(outputImageName,
      os.O_WRONLY|os.O_TRUNC|os.O_CREATE,0600,)
    if err != nil {
        panic(err)
    }

    // Save the picture !
    fmt.Println("Encoding to PNG...")
    png.Encode(outputFile, outputImage)
    fmt.Printf("Writing a picture to %s\n", outputImageName)
    outputFile.Close()

    // ...
    if (!loopAgain) {
      break
    }

  } // While Part(s)
  fmt.Println("Done !")
}

func decodeFile(files []string, destination string) {

  // Find the original name
  name := path.Base(files[0])
  baseName := name[0:strings.LastIndex(name, ".GooglePhotosStorage")]
  outputFileName := destination + "/" + baseName

  // Open the destination file and truncate it
  f, err := os.OpenFile(outputFileName, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0600)
  if err != nil {
      panic(err)
  }
  f.Write([]byte{})
  f.Close()

  // Useful vars
  lastIndexFile := len(files) - 1

  // For each picture to decode
  for indexFile, file := range files {

    fmt.Printf("Opening %s for decoding...\n", file)
    inputFile, err := os.Open(file)
    if err != nil {
        panic(err)
    }
    defer inputFile.Close()

    inputImage, _, err := image.Decode(inputFile)
    if err != nil {
        log.Fatal("Error while decoding the file. Is your file an PNG image created by this tool?")
    }

    // Picture to array
    inputImageBounds := inputImage.Bounds()
    inputImageD := image.NewNRGBA64(inputImageBounds)
    draw.Draw(inputImageD, inputImageBounds, inputImage, inputImageBounds.Min, draw.Src)
    inputBuffer := inputImageD.Pix
    
    if (lastIndexFile == indexFile) { // We need to check the EOF
      inputBufferCleaned := make([]byte, 0, len(inputBuffer))
      for index, value := range inputBuffer {
        if value == endMarker[0] && bytes.Equal(inputBuffer[index:index+64], endMarker) {
          break
        } else {
          inputBufferCleaned = append(inputBufferCleaned, value)
        }
      }
      inputBuffer = inputBufferCleaned
    } else {
      // Write the whole file
      inputBuffer = inputBuffer[:maxGooglePhotoImageSize]
    }

    // Re-open the file in append mode
    f, err := os.OpenFile(
        outputFileName,
        os.O_WRONLY| os.O_APPEND,
        0600,
    )
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Appending data...")
    _, err = f.Write(inputBuffer)
    if err != nil {
        log.Fatal(err)
    }
    f.Close()

  } // End of loop encoded files
  fmt.Printf("The file %s if now ready !\n", outputFileName)
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

// Main

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
