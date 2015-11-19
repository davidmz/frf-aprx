# frf-aprx

Прокси для передачи авторизационного токена на freefeed.net через параметр запроса. Написан для работы с фрифидом через Maker IFTTT (https://ifttt.com/maker).

Распознаёт запросы с Content-Type `application/x-www-form-urlencoded` или `application/json`. В первом случае авторизационный токен берётся из поля `accessToken` POST-формы, во втором — из поля `accessToken` посылаемого объекта. В обоих случаях соответствующее поле из запроса удаляется, и запрос проксируется на freefeed.net с установленным заголовком `X-Authentication-Token`.