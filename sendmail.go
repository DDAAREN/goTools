package goTools

import (
    "crypto/tls"
    "fmt"
    "net"
    "net/mail"
    "net/smtp"
    "log"
)

const (
    servername = "smtp.xxx.com:25"
    username = "xxx@xxx.com"
    password = "xxxxx"
)

func dial(addr string) (*tls.Conn, error) {
    return tls.Dial("tcp", addr, nil)
}

func composeMsg(from string, to string, subject string, body string) (message string) {
    headers := make(map[string]string)
    headers["From"] = from
    headers["To"] = to
    headers["Subject"] = subject

    for k,v := range headers {
        message += fmt.Sprintf("%s: %s\r\n", k, v)
    }
    message += "\r\n" + body

    return 
}

func SendMail(toAddr string, subject string, body string) (err error) {
    host,_,_ := net.SplitHostPort(servername)
    auth := smtp.PlainAuth("",username, password, host)

    from := mail.Address{"",username}
    to := mail.Address{"",toAddr}

    message := composeMsg(from.String(), to.String(), subject, body)

    err = smtp.SendMail(
        servername,
        auth,
        username,
        []string{toAddr},
        []byte(message),
    )

    if err != nil {
        log.Printf("smtp error: %s", err)
        return err
    }

    log.Print("Sent.")
    return nil
}

func SendMail_SSL(toAddr string, subject string, body string) (err error) {
    host,_,_ := net.SplitHostPort(servername)
    conn, err := dial(servername)
    if err != nil {
        log.Println(err)
        return err
    }

    smtpClient, err := smtp.NewClient(conn, host)
    if err != nil {
        log.Println(err)
        return err
    }

    auth := smtp.PlainAuth("",username, password, host)

    err = smtpClient.Auth(auth)
    if err != nil {
        log.Println(err)
        return err
    }

    from := mail.Address{"",username}
    to := mail.Address{"",toAddr}
    err = smtpClient.Mail(from.Address)
    if err != nil {
        log.Println(err)
        return err
    }

    err = smtpClient.Rcpt(to.Address)
    if err != nil {
        log.Println(err)
        return err
    }

    writer, err := smtpClient.Data()
    if err != nil {
        log.Println(err)
        return err
    }

    message := composeMsg(from.String(), to.String(), subject, body)
    

    _, err = writer.Write([]byte(message))
    if err != nil {
        log.Println(err)
        return err
    }

    err = writer.Close()
    if err != nil {
        log.Println(err)
        return err
    }

    smtpClient.Quit()

    fmt.Print("Send..")
    return nil
}
