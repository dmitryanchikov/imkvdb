package storage

// Storage – это верхнеуровневый интерфейс для работы с ключ-значение хранилищем.
// В реальном приложении он мог бы содержать больше методов (Init, Close, Backup и т.д.).
type Storage interface {
	Set(key, value string) error
	Get(key string) (string, bool)
	Del(key string) bool
}
