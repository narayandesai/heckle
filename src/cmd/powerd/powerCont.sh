#!/usr/bin/expect
#log_user 0
set address [lindex $argv 0]
set username [lindex $argv 1]
set password [lindex $argv 2]
set operation [lindex $argv 3]
set outlet [lindex $argv 4]
spawn telnet ${address}
expect {*Username:}
send "${username}\r"
expect {*Password:}
send "${password}\r"
expect {*Switched CDU: }
send "${operation} ${outlet}\r"
expect {*Switched CDU: }
send "exit\r"