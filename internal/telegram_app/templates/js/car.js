

let tg = window.Telegram.WebApp;
let text = "RENT";

tg.MainButton.setText(text);
tg.MainButton.show();
tg.MainButton.enable()

tg.MainButton.requestContact()