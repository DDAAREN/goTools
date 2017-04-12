package goTools

import (
    "log"
    "net/smtp"
)

func SendMail(subject string, body string) {
    from := "18301109200@163.com"
    pass := "163mailbot"
    to := "liulei@btime.com"
    
    msg := "From: " + from + "\n" +
            "To: " + to + " \n" +
            "Subject: " + subject +"\n\n" +
            body

    auth := smtp.PlainAuth("",from,pass,"smtp.163.com")

    err := smtp.SendMail(
        "smtp.163.com:25",
        auth,
        from,
        []string{to},
        []byte(msg),
    )

    if err != nil {
        log.Printf("smtp error: %s", err)
        return
    }

    log.Print("Sent.")
}

func SendMail_raw(subject string, body []byte) {
    from := "18301109200@163.com"
    pass := "163mailbot"
    to := "liulei@btime.com"
    
    auth := smtp.PlainAuth("",from,pass,"smtp.163.com")

    err := smtp.SendMail(
        "smtp.163.com:25",
        auth,
        from,
        []string{to},
        body,
    )

    if err != nil {
        log.Printf("smtp error: %s", err)
        return
    }

    log.Print("Sent.")
}

