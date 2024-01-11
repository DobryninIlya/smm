let tg = window.Telegram.WebApp;
let text = "RENT";

alert(tg.initDataUnsafe.user.username);
tg.MainButton.setText(text);
tg.MainButton.show();
tg.MainButton.enable();

// tg.MainButton.requestContact()