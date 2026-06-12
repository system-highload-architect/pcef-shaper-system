# 📱 User Equipment (UE) Emulator — Technical Specification (SRS)

### 📋 1. Core Objectives / Основные бизнес-цели
* **[RU]** Написание легковесного многопоточного стресс-тестировщика (Traffic Generator) на горутинах для генерации пиковой Highload-нагрузки на наше PCEF-ядро для проверки лимитов.
* **[EN]** Development of a lightweight, highly-concurrent traffic stress-tester (Traffic Generator) powered by goroutines to unleash peak Highload bursts on our PCEF shaper.

### ⚙️ 2. Algorithmic Logic & Boundary Conditions / Логика вычислений и пограничные условия
* **[RU]** Каждая запущенная горутина `G` представляет собой отдельный смартфон абонента [🧠]. Внутри крутится бесконечный цикл, который с кастомным джиттером времени (`time.Sleep`) шлет в наш шлюз пакеты `NetworkPacketFrame` со случайными хостами (`youtube.com`, `telegram.org`) и размерами пакетов от 64 байт до 1.5 Мегабайт [🧠]. Это позволяет проверить работу и хэш-таблиц DPI, и бинарного поиска диапазонов [🧠].
* **[EN]** Each spawned goroutine `G` mimics a unique subscriber device. Inside, an infinite loop drives transaction bursts with a randomized jitter delay (`time.Sleep`), firing `NetworkPacketFrame` structures populated with random target hosts (`youtube.com`, `telegram.org`) and packet payload scales from 64 bytes to 1.5 Megabytes. This stress-tests both DPI hash maps and range binary search trees.

### 🎖️ 3. Acceptance Criteria / Критерии приемки кода
1. Способность генератора без труда масштабироваться до 10 000 параллельных горутин-абонентов на стандартном ноутбуке [🧠].
2. Логика генерации хостов покрывает как валидные сигнатуры тарифов, так и неизвестный `GENERIC` трафик [🧠].
