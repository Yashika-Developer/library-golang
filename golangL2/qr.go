package main

import (
    "bytes"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/mysql"
    "github.com/skip2/go-qrcode"
)

var db *gorm.DB
var err error

type BookInventory struct {
    ISBN            string `gorm:"primary_key"`
    LibID           uint
    Title           string `json:"title"`
    Authors         string `json:"authors"`
    Publisher       string `json:"publisher"`
    Version         string `json:"version"`
    TotalCopies     int    `json:"total_copies"`
    AvailableCopies int    `json:"available_copies"`
}

func ConnectDb() (*gorm.DB, error) {
    db, err = gorm.Open("mysql", "root:12345@tcp(127.0.0.1:3306)/MYDB?charset=utf8mb4&parseTime=True")
    if err != nil {
        return nil, err
    }
    return db, nil
}

func GenerateQRCode(c *gin.Context) {
    isbn := c.Param("isbn")

    // Fetch book details from the database
    var book BookInventory
    if err := db.Where("isbn = ?", isbn).First(&book).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
        return
    }

    // Generate QR code with book details
    qrData := book.Title + "\n" + book.Authors + "\n" + book.Publisher + "\n" + book.Version
    qr, err := qrcode.Encode(qrData, qrcode.Medium, 256)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
        return
    }

    // Serve QR code as image
    c.Data(http.StatusOK, "image/png", qr)
}

func main() {
    // Connect to the database
    db, err := ConnectDb()
    if err != nil {
        panic("failed to connect database")
    }
    defer db.Close()

    // Initialize Gin router
    router := gin.Default()

    // Route to generate QR code for a book
    router.GET("/generate-qr/:isbn", GenerateQRCode)

    // Run the server
    router.Run(":8080")
}



<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>QR Code Generator</title>
  <script src="https://cdn.jsdelivr.net/npm/qrcode"></script>
</head>
<body>
  <h1>QR Code Generator</h1>
  <label for="isbn">Enter ISBN:</label>
  <input type="text" id="isbn" name="isbn" placeholder="Enter ISBN...">
  <button onclick="generateQR()">Generate QR Code</button>
  <div id="qrcode"></div>

  <script>
    function generateQR() {
      var isbn = document.getElementById('isbn').value;
      if (!isbn) {
        alert('Please enter an ISBN');
        return;
      }

      // Send ISBN to backend to generate QR code
      fetch(`/generate-qr?isbn=${isbn}`)
        .then(response => response.json())
        .then(data => {
          if (data.error) {
            alert(data.error);
          } else {
            displayQR(data.qrCode);
          }
        })
        .catch(error => {
          console.error('Error:', error);
          alert('Failed to generate QR code');
        });
    }

    function displayQR(qrCode) {
      var qrCodeDiv = document.getElementById('qrcode');
      qrCodeDiv.innerHTML = qrCode;
    }
  </script>
</body>
</html>
