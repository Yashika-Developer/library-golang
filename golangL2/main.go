package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var db *gorm.DB
var err error

type Library struct {
	ID   uint   `gorm:"primary_key"`
	Name string `json:"name"`
}

type User struct {
	ID            uint   `gorm:"primary_key"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	ContactNumber string `json:"contact_number"`
	Role          string `json:"role"`
	LibID         uint   `json:"lib_id"`
}

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

type RequestEvent struct {
	ReqID        uint   `gorm:"primary_key"`
	BookID       string `json:"book_id"`
	ReaderID     uint   `json:"reader_id"`
	RequestDate  string `json:"request_date"`
	ApprovalDate string `json:"approval_date"`
	ApproverID   uint   `json:"approver_id"`
	RequestType  string `json:"request_type"`
}

type IssueRegistry struct {
	IssueID            uint   `gorm:"primary_key"`
	ISBN               string `json:"isbn"`
	ReaderID           uint   `json:"reader_id"`
	IssueApproverID    uint   `json:"issue_approver_id"`
	IssueStatus        string `json:"issue_status"`
	IssueDate          string `json:"issue_date"`
	ExpectedReturnDate string `json:"expected_return_date"`
	ReturnDate         string `json:"return_date"`
	ReturnApproverID   uint   `json:"return_approver_id"`
}

func ConnectDb() (*gorm.DB, error) {
	db, err = gorm.Open("mysql", "root:12345@tcp(127.0.0.1:3306)/MYDB?charset=utf8mb4&parseTime=True")
	if err != nil {
		log.Println("Connection Failed to Open")
	} else {
		log.Println("Connection Established")
	}
	return db, err
}

func CreateLibrary(c *gin.Context) {
	var req struct {
		LibraryName string `json:"library_name"`
		UserName    string `json:"user_name"`
		UserEmail   string `json:"user_email"`
		UserContact string `json:"user_contact"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the user email already exists
	var existingUser User
	if err := db.Where("email = ?", req.UserEmail).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User with the same email already exists"})
		return
	}

	// Check if the library already exists
	var existingLibrary Library
	if err := db.Where("name = ?", req.LibraryName).First(&existingLibrary).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Library with the same name already exists"})
		return
	}

	// Create the library
	newLibrary := Library{Name: req.LibraryName}
	if err := db.Create(&newLibrary).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create library"})
		return
	}

	// Add the user as owner
	newUser := User{
		Name:          req.UserName,
		Email:         req.UserEmail,
		ContactNumber: req.UserContact,
		Role:          "Owner",
		LibID:         newLibrary.ID,
	}
	if err := db.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user as owner"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Library created successfully"})
}

func AddLibraryAdmin(c *gin.Context) {
	var req struct {
		LibID         uint   `json:"lib_id"`
		Name          string `json:"name"`
		Email         string `json:"email"`
		ContactNumber string `json:"contact_number"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the provided library ID exists
	var library Library
	if err := db.First(&library, req.LibID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Library not found"})
		return
	}

	// Check if the provided email already exists
	var existingUser User
	if err := db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User with the same email already exists"})
		return
	}

	// Create the library admin
	newAdmin := User{
		Name:          req.Name,
		Email:         req.Email,
		ContactNumber: req.ContactNumber,
		Role:          "LibraryAdmin",
		LibID:         req.LibID,
	}
	if err := db.Create(&newAdmin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create library admin"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Library admin created successfully"})
}

func AddBooks(c *gin.Context) {
	var book BookInventory

	if err := c.BindJSON(&book); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Check if TotalCopies or AvailableCopies is negative
	if book.TotalCopies < 0 || book.AvailableCopies < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TotalCopies and AvailableCopies cannot be negative"})
		return
	}
	// Set LibID to 1
	book.LibID = 1

	// Check if the book already exists
	var existingBook BookInventory
	if err := db.Where("isbn = ?", book.ISBN).First(&existingBook).Error; err == nil {
		// If the book exists, update its total copies
		existingBook.TotalCopies += book.TotalCopies
		existingBook.AvailableCopies += book.TotalCopies // Increase available copies as well
		if err := db.Save(&existingBook).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update book copies"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Book updated successfully"})
		return
	}

	// If the book doesn't exist, create a new entry
	book.AvailableCopies = book.TotalCopies // Set available copies equal to total copies
	if err := db.Create(&book).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add book"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Book added successfully"})
}
func RemoveBook(c *gin.Context) {
	isbn := c.Param("isbn")

	// JSON request structure
	var req struct {
		CopiesToRemove int `json:"copies_to_remove"`
	}

	// Bind JSON request
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the book exists
	var existingBook BookInventory
	if err := db.Where("isbn = ?", isbn).First(&existingBook).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	// Prompt the user for the number of copies to remove
	if req.CopiesToRemove <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid number of copies to remove"})
		return
	}

	if req.CopiesToRemove > existingBook.TotalCopies {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Requested copies to remove exceed total copies"})
		return
	}

	// Update available copies
	existingBook.AvailableCopies -= req.CopiesToRemove

	// Update total copies
	existingBook.TotalCopies -= req.CopiesToRemove

	// Save changes to the database
	if err := db.Save(&existingBook).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove book"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Book removed successfully"})
}

func ApproveRejectRequest(c *gin.Context) {
	reqID := c.Param("req_id")
	action := c.Query("action")

	// Fetch the request from the database
	var request RequestEvent
	if err := db.First(&request, reqID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	// Determine if the action is to approve or reject
	switch action {
	case "approve":
		// Set the approval date and approver ID
		request.ApprovalDate = time.Now().Format("2006-01-02")
		// Assuming the approver ID is the admin's user ID
		// Get the admin's user ID from the email provided in the request
		email, _ := c.Get("email")
		var admin User
		db.Where("email = ?", email.(string)).First(&admin)
		request.ApproverID = admin.ID

		// Update the request status in the database
		if err := db.Save(&request).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve request"})
			return
		}

		// Update the issue registry accordingly
		issue := IssueRegistry{
			ISBN:               request.BookID,
			ReaderID:           request.ReaderID,
			IssueApproverID:    request.ApproverID,
			IssueStatus:        "Approved",
			IssueDate:          time.Now().Format("2006-01-02"),
			ExpectedReturnDate: time.Now().AddDate(0, 0, 7).Format("2006-01-02"), // Assuming 7 days from today
		}
		if err := db.Create(&issue).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update issue registry"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Request approved successfully"})

	case "reject":
		// Set the request status to rejected
		request.RequestType = "Rejected"

		// Update the request status in the database
		if err := db.Save(&request).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject request"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Request rejected successfully"})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action. Please specify 'approve' or 'reject'"})
	}
}

func ListIssueRequests(c *gin.Context) {
	var issueRequests []RequestEvent

	// Get the email of the logged-in user from the request header
	adminEmail, _ := c.Get("email")

	// Fetch the admin from the database
	var admin User
	if err := db.Where("email = ?", adminEmail).First(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch admin"})
		return
	}

	// Fetch all pending issue requests in the admin's library
	if err := db.Where("lib_id = ? AND request_type = ? AND approval_date IS NULL", admin.LibID, "Issue").Find(&issueRequests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch issue requests"})
		return
	}

	c.JSON(http.StatusOK, issueRequests)
}

func UpdateBookDetails(c *gin.Context) {
	var book BookInventory

	if err := c.BindJSON(&book); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the book exists
	var existingBook BookInventory
	if err := db.Where("isbn = ?", book.ISBN).First(&existingBook).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	// Update the book details
	if err := db.Model(&existingBook).Updates(&book).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update book details"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Book details updated successfully"})
}

func RaiseIssueRequest(c *gin.Context) {
	var req struct {
		BookID   string `json:"book_id"`
		ReaderID uint   `json:"reader_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check book availability
	var book BookInventory
	if err := db.Where("isbn = ?", req.BookID).First(&book).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}
	var reader User
	if err := db.Where("id = ?", req.ReaderID).First(&reader).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "reader not found"})
		return
	}
	if reader.Role != "reader" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid role for this user"})
		return
	}

	if book.AvailableCopies == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Book is not available"})
		return
	}

	// Create issue request
	requestEvent := RequestEvent{
		BookID:      req.BookID,
		ReaderID:    req.ReaderID,
		RequestDate: time.Now().Format("2006-01-02"),
		RequestType: "Issue", // Assuming this is how you differentiate issue requests
	}
	if err := db.Create(&requestEvent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to raise issue request"})
		return
	}

	// Decrement available copies
	if err := db.Model(&book).Update("available_copies", book.AvailableCopies-1).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update available copies"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Issue request raised successfully"})
}
func SearchBook(c *gin.Context) {
	var req struct {
		Title     string `json:"title"`
		Author    string `json:"author"`
		Publisher string `json:"publisher"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var books []BookInventory
	query := db.Where("title LIKE ? OR authors LIKE ? OR publisher LIKE ?", "%"+req.Title+"%", "%"+req.Author+"%", "%"+req.Publisher+"%")
	if err := query.Find(&books).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search for books"})
		return
	}

	// Check availability and calculate expected due date
	for i, book := range books {
		if book.AvailableCopies == 0 {
			// Book is not available, calculate expected due date
			expectedDueDate := time.Now().AddDate(0, 0, 7).Format("2006-01-02") // Assuming 7 days from today
			books[i].AvailableCopies = 0
			books[i].Version = expectedDueDate
			c.JSON(http.StatusOK, gin.H{"book is not avalable and expected due date": expectedDueDate})

		}

	}
}
func RegisterUser(c *gin.Context) {
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user.Role = "reader" // Set role as reader

	// Connect to the database
	db, err := ConnectDb()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to database"})
		return
	}
	defer db.Close()

	// Create the user in the database
	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

func Login(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch the user from the database
	var user User
	result := db.Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Check if the user's role matches the selected role
	if req.Role != user.Role {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid role for this user"})
		return
	}

	// Determine if the user is an admin or a reader
	isAdmin := user.Role == "LibraryAdmin"
	isReader := user.Role == "reader"

	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"role":     user.Role,
		"isAdmin":  isAdmin,
		"isReader": isReader,
	})
}

func main() {
	db, err := ConnectDb()
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()
	db.AutoMigrate(&Library{})
	db.AutoMigrate(&User{})
	db.AutoMigrate(&BookInventory{})
	db.AutoMigrate(&RequestEvent{})
	db.AutoMigrate(&IssueRegistry{})

	router := gin.Default()
	// Enable CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}
		c.Next()
	})
	router.POST("/login", Login)
	router.POST("/library", CreateLibrary)
	router.POST("/library-admin", AddLibraryAdmin)
	router.POST("/register-reader", RegisterUser)
	router.POST("/add-book", AddBooks)
	router.POST("/remove-book/:isbn", RemoveBook)
	router.POST("/update-book", UpdateBookDetails)
	router.POST("/search-book", SearchBook)
	router.POST("/raise-issue", RaiseIssueRequest)
	router.GET("/list-issue-requests", ListIssueRequests)
	router.POST("/approve-reject-request/:req_id", ApproveRejectRequest)
	router.Run("localhost:8080")
}
