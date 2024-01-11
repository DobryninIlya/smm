let tg = window.Telegram.WebApp;
let text = "RENT";

console.log(tg.initDataUnsafe.user.username);
console.log(tg.MainButton.setText(text));
console.log(tg.MainButton.show());
console.log(tg.MainButton.enable());

tg.MainButton.requestContact();