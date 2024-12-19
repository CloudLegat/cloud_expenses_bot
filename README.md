Cloud Expenses Bot

Cloud Expenses Bot is a simple Telegram bot designed to help you track your daily expenses using Google Sheets. Initially created for personal use, I've decided to share it to assist others in managing their budgets effortlessly.

🚀 Features

    Add Expenses: Log your expenses with easy commands.
    View Daily Budget: Check your remaining budget for the day.
    Multi-language Support: Use the bot in English or Russian.
    Google Sheets Integration: All data is securely stored in your Google Sheets.

🔧 Setup

    Clone the Repository:

git clone https://github.com/yourusername/cloud-expenses-bot.git
cd cloud-expenses-bot

Configure:

    Fill in your details in config.json:
        spreadsheet_id: Your Google Sheets ID.
        bot_token: Your Telegram bot token.
        cell_ranges: Define your Google Sheets cell ranges.
    
    Ensure lang.json contains your desired languages.

Install Dependencies:

go get -u github.com/go-telegram-bot-api/telegram-bot-api/v5
go get -u google.golang.org/api/sheets/v4
go get -u golang.org/x/oauth2/google

Run the Bot:

    go run main.go

Follow the console instructions to authorize with Google.

🛠 Usage

    Start the Bot:

/start

Add an Expense:

/add AMOUNT [n/k] CATEGORY

    AMOUNT: Number (e.g., 150)
    [n/k]: n for cash, k for card
    CATEGORY: Expense category (e.g., food)

Example:

/add 400 n home

View Daily Budget:

/budget

Change Language:

/lang en

or

    /lang ru

📁 Files

    main.go: Main application code.
    lang.json: Language strings.
    config.json: Configuration settings.

🔒 Security

    Keep credentials.json and token.json secure.
    Add them to .gitignore to prevent accidental commits.

📄 License

This project is licensed under the MIT License.