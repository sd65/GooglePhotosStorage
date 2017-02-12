package main

import (
    "os"
    "log"
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

const maxGooglePhotoImageSize int = 70 * 1024 * 1024
var endMarker = []byte("<<••—{Th¡sIs±The†For" + "GooglePhotosStorage…}—••>>") // 8 bytes

type args struct {
  Encode string `arg:"-e,help: encode file to PNG"`
  Decode []string `arg:"-d,help: decode file from PNG"`
  Destination string
}

func encodeFile(inputFile string, destination string) {

  // Max size for Google Photos
  const maxWidth int = 4614
  const maxHeight int = 3464

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
  var writeIndexMarker int = 0
  var bytesRead int = 0

  for {

    loopAgain := false

    // Create the image object
    outputImage := image.NewNRGBA64(image.Rect(0, 0, maxWidth, maxHeight))

    // Vars for loop
    x := 0
    maxX:= outputImage.Bounds().Max.X
    y := 0

    // Loop
    for {
      if bytesRead + 8 - (maxGooglePhotoImageSize * part) > maxGooglePhotoImageSize {
        loopAgain = true
        break
      }
      count, _ := inputFileReader.Read(buf)
      bytesRead += count
      for i := count; i < 8; i ++ {
        // EOF, complete with end marker
        buf[i] = endMarker[writeIndexMarker]
        writeIndexMarker++
        if (writeIndexMarker == 64) {
          writeIndexMarker = -1
          break
        }
      }
      pixelColor := color.NRGBA64{
        binary.BigEndian.Uint16(buf[0:2]),
        binary.BigEndian.Uint16(buf[2:4]),
        binary.BigEndian.Uint16(buf[4:6]),
        binary.BigEndian.Uint16(buf[6:8]),
      }
      // Set it
      outputImage.Set(x, y, pixelColor)
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

    // Choose the name
    part += 1
    var outputImageName string
    if (!loopAgain && part == 1) {
      outputImageName = outputImageBaseName + ".png"
    } else {
      outputImageName = outputImageBaseName + ".part" + strconv.Itoa(part) + ".png"
    }

    // The file (part) destination
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
  var bytesWritten int = 0


  for indexFile, file := range files {

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
      inputBuffer = inputBuffer[:maxGooglePhotoImageSize]
    }

    f, err := os.OpenFile(
        outputFileName,
        os.O_WRONLY| os.O_APPEND,
        0600,
    )
    if err != nil {
        log.Fatal(err)
    }
    tmpBytesWritten, err := f.Write(inputBuffer)
    if err != nil {
        log.Fatal(err)
    }
    f.Close()
    bytesWritten += tmpBytesWritten


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
