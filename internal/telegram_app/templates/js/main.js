var SearchResultBlock = document.getElementById("searchResultBlock");

var SelectedGetCity = "";
var SelectedReturnCity = "";
var SelectedGetDate = "";
var SelectedReturnDate = "";
var SelectedTabCarType = "legkovoj-transport";
var SelectedMinPrice = 0;
var SelectedMaxPrice = 2147483647;

let today = new Date().toISOString().substr(0, 10);
document.querySelector("#getDateInput").value = today;

let future = new Date();
future.setDate(future.getDate() + 7);
document.querySelector("#returnDateInput").value = future.toISOString().substr(0, 10);


function selectTab(tab, type) {
    // Удаляем активный класс у всех вкладок
    const tabs = document.querySelectorAll('.category_tab');
    tabs.forEach(function (tab) {
        tab.classList.remove('active');
    });

    // Добавляем активный класс только выбранной вкладке
    tab.classList.add('active');
    SelectedTabCarType = type;
}


async function loadCities() {
    const response = await fetch('/static/json/cities.json');
    const citiesData = await response.json();
    return citiesData;
}

function closeModal() {
    var modal = document.querySelector('.modal');
    modal.style.display = 'none';
}

// Функция для добавления вариантов поиска
function addSearchOptions(inputId, suggestionsId, data) {
    var input = document.getElementById(inputId);
    var suggestions = document.getElementById(suggestionsId);

    input.addEventListener("input", function () {
        var inputValue = input.value.toLowerCase();
        suggestions.innerHTML = "";

        data.cities.forEach(function (city) {
            var cityName = city.cityName.toLowerCase();
            var location = city.location.toLowerCase();

            if (cityName.includes(inputValue) || location.includes(inputValue)) {
                var li = document.createElement("li");
                li.textContent = cityName + " (" + location + ")";
                li.addEventListener("click", function () {
                    input.value = cityName;
                    suggestions.innerHTML = "";
                    console.log("Slug:", city.slug); // Выводим slug в консоль
                    if (inputId === "getCityInput") {
                        SelectedGetCity = city.slug;
                        SelectedReturnCity = city.slug;
                        document.getElementById("returnCityInput").value = cityName;
                    }
                    if (inputId === "returnCityInput") {
                        SelectedReturnCity = city.slug;
                    }
                });
                suggestions.appendChild(li);
            }
        });
    });
}

// Загружаем города и добавляем варианты поиска для поля "Получение"
loadCities().then(function (citiesData) {
    addSearchOptions("getCityInput", "getCitySuggestions", citiesData);
});

// Загружаем города и добавляем варианты поиска для поля "Возврат"
loadCities().then(function (citiesData) {
    addSearchOptions("returnCityInput", "returnCitySuggestions", citiesData);
});


function SendDataForm() {
    SelectedGetDate = document.getElementById("getDateInput").value;
    SelectedReturnDate = document.getElementById("returnDateInput").value;
    SelectedMinPrice = document.getElementById("costFromInput").value;
    SelectedMaxPrice = document.getElementById("costToInput").value;
    var url = "/app/get_cars?pickup="+SelectedGetDate+"&drop="+SelectedReturnDate+"&transport="+SelectedTabCarType+"&location_slug="+SelectedGetCity+"&drop_city="+SelectedReturnCity+
        "&min_price="+SelectedMinPrice+"&max_price="+SelectedMaxPrice;
    console.log(url);
    ProcessQuery(url);
}

searchButton = document.getElementById("searchButton");
searchButton.addEventListener("click", SendDataForm);

function ProcessQuery(url) {
    fetch(url)
        .then(response => response.text())
        .then(html => {
            SearchResultBlock.innerHTML = html
        })
}


var catalogList = document.querySelector(".catalog-list");

if (catalogList) {
    catalogList.addEventListener("click", function(event) {
        var clickedElement = event.target.closest('.catalog-item');

        if (clickedElement) {
            event.preventDefault();

            var linkElement = clickedElement.querySelector(".catalog-item_link");
            var link = linkElement ? linkElement.getAttribute("href") : null;

            if (link) {
                openModal(link);
            }
        }
    });
}

function openModal(link) {
    // Создаем элемент модального окна
    var modal = document.createElement("div");
    modal.classList.add("modal");

    // Создаем iframe для загрузки содержимого страницы
    var iframe = document.createElement("iframe");
    iframe.src = "/app/rent?link=" + encodeURIComponent(link);
    iframe.style.width = "100%";
    iframe.style.height = "100%";
    iframe.style.border = "none";


    // Добавляем iframe в модальное окно
    modal.innerHTML = '<span class="close .options-style1_item" onclick="closeModal()">&times;</span>'
    modal.appendChild(iframe);

    // Добавляем модальное окно в body
    document.body.appendChild(modal);

    // Добавляем обработчик события для закрытия модального окна
    modal.addEventListener("click", function() {
        document.body.removeChild(modal);
    });
}