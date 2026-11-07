// Package ggsel — неофициальный клиент Seller API GGSel.
//
// # Быстрый старт
//
// [New] собирает клиент. Пустой ключ — ошибка. Из New сеть не вызывается.
// Дедлайн запроса задаётся только через context.
//
// # Контур v2
//
// Заголовок Authorization равен ключу, без Bearer.
// Методы: [Client.ListOffers], [Client.GetOffer], [Client.CreateOffer], [Client.UpdateOffer],
// [Client.ListCategories] и [Client.SearchCategories].
//
// # Контур v1
//
// [Client.Login] необязателен. Token кэшируется до valid_thru и в методы не передаётся.
// Без SellerID методы v1 возвращают [ErrSellerIDRequired].
// Методы: [Client.GetOrder], [Client.CheckUniqueCode], [Client.ListChats], [Client.ListMessages], [Client.SendMessage].
//
// # Деньги
//
// [Decimal] хранит цены и суммы. float64 не используется. В запросе цена передаётся указателем.
//
// # Частичное обновление
//
// Nil-поле не сериализуется. [Ptr] нужен, чтобы отправить false, 0 или пустую строку.
//
// # Ошибки
//
// [APIError] несёт HTTP-статус и разобранное тело. [IsUnauthorized] сообщает статус 401.
// errors.As находит APIError. Error не содержит ключ, sign и token.
//
// # Повторы
//
// Повторяются только помеченные безопасные GET. 401 и отказ apilogin не повторяются.
//
// # Чего нет
//
// Нет batch и async job, options, products, прочих методов v1 и XML.
// Это неофициальный снимок публичной документации, не спецификация GGSel.
package ggsel
