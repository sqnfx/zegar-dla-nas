# zegar nie goń się (AI/SI/KI etc.)

pobierz oraz w środku:
go build -trimpath -ldflags "-H windowsgui -s -w" -o ZegarPasek.exe .

Jak działa: zatrzymaj kursor na pasku na ułamek sekundy, a obok pojawi się okrągłe okienko ze szklaną obwódką. Idzie za kursorem wzdłuż paska i znika, gdy z niego zjedziesz.
