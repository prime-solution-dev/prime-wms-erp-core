package emailservice

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"prime-erp-core/internal/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/gomail.v2"
)

type ConfigEmailAlert struct {
	Host       string   `json:"host"`
	Port       int      `json:"port"`
	User       string   `json:"user"`
	Password   string   `json:"password"`
	Sender     string   `json:"sender"`
	Recipients []string `json:"recipients"`
}

func SendEmailAlertForNewBrand(ctx *gin.Context, jsonPayload string) (interface{}, error) {

	var req []models.CreditRequest

	if err := json.Unmarshal([]byte(jsonPayload), &req); err != nil {
		return nil, errors.New("failed to unmarshal JSON into struct: " + err.Error())
	}

	host := os.Getenv("email_host")
	portStr := os.Getenv("email_port")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid port: %v", err)
	}
	user := os.Getenv("email_user")
	password := os.Getenv("email_password")
	sender := os.Getenv("email_sender")
	recipients := []string{"champsamui8@gmail.com"}
	subject := "Credit Limit Exceeded" + time.Now().AddDate(0, 0, -1).Format("02/01/2006")

	var bodyRows string
	for _, creditRequestValue := range req {
		bodyContent := fmt.Sprintf(`
		 
			<p>Customer Code : "%s"</p>
			<p>Customer Name : "%s"</p>
			<p>Credit limit (THB) : "%.2f"</p>
			<p>Increase Credit limit (THB) : "%.2f"</p>
			<p>Total Credit limit (THB) : "%.2f"</p>
			<p>Consumed Credit : "%.2f"</p>
			<p>Balance Credit Limit :  "%.2f"</p>
		 	<br/>
	`, creditRequestValue.CustomerCode, creditRequestValue.CustomerName, creditRequestValue.Amount, creditRequestValue.TemporaryIncreaseCreditLimit,
			creditRequestValue.Amount+creditRequestValue.TemporaryIncreaseCreditLimit, creditRequestValue.ConsumedCredit, creditRequestValue.BalanceCreditLimit)
		bodyRows += bodyContent
	}

	bodyContent := fmt.Sprintf(`
		<html>
		<body>
			 "%s"
		</body>
		</html>
	`, bodyRows)

	m := gomail.NewMessage()
	m.SetHeader("From", sender)
	m.SetHeader("To", recipients...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", bodyContent)
	d := gomail.NewDialer(host, port, user, password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	if err := d.DialAndSend(m); err != nil {
		log.Printf("Could not send email: %v", err)
		return nil, err
	}

	log.Println("Email Alert ForNewBrand sent successfully!")
	return nil, nil
}
