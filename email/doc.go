package email

// email is responsible for sending email to an SMTP relay, including
// connecting to the server, negotiating TLS and authentication, and building
// a MIME-formatted email body. It is not designed to represent the user-facing
// content of an email, and includes this content in email bodies regardless
// of what it contains.
