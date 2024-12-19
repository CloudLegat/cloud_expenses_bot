# Cloud Expenses Bot

Cloud Expenses Bot is a personal budget management bot that integrates with Google Sheets and Telegram. This bot helps you track your daily expenses, manage your budget, and provides support for multiple languages (English and Russian).

## Features

- Track daily expenses
- Categorize expenses
- View daily budget
- Support for multiple languages (English and Russian)
- Integration with Google Sheets and Telegram

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/yourusername/cloud-expenses-bot.git
    cd cloud-expenses-bot
    ```

2. Install dependencies:
    ```sh
    go mod tidy
    ```

3. Create a `config.json` file with the following structure:
    ```json
    {
        "spreadsheet_id": "your_google_sheet_id",
        "bot_token": "your_telegram_bot_token",
        "cell_ranges": {
            "daily_expenses": "I2:I32",
            "category_range": "A2:A32",
            "category_column": "B",
            "budget_column": "C"
        }
    }
    ```

4. Create a `credentials.json` file with your Google API credentials. [How to create credentials](https://developers.google.com/sheets/api/quickstart/go).

5. Create a `lang.json` file with the following structure:
    ```json
    {
        "en": {
            "start": "Welcome to Cloud Expenses Bot!",
            "add_usage": "Usage: /add <amount> <card|cash> <category>",
            "invalid_amount": "Invalid amount format.",
            "invalid_suffix": "Invalid payment method. Use 'card' or 'cash'.",
            "expense_added": "Added %.2f to %s via %s.",
            "daily_budget": "Your daily budget is: %.2f",
            "error_occurred": "An error occurred: %v",
            "unknown_command": "Unknown command.",
            "help_message": "This bot helps you manage your expenses.",
            "suffix_card": "card",
            "suffix_cash": "cash",
            "payment_card": "card",
            "payment_cash": "cash",
            "category_not_found": "Category %s not found.",
            "language_set": "Language set to %s.",
            "select_language": "Please select a language: en or ru."
        }
    }
    ```

## Usage

1. Run the bot:
    ```sh
    go run main.go
    ```

2. Interact with the bot on Telegram using the following commands:
    - `/start` - Start the bot and get a welcome message.
    - `/add <amount> <card|cash> <category>` - Add an expense.
    - `/budget` - Get your daily budget.
    - `/lang <en|ru>` - Set your preferred language.
    - `/help` - Get help information.

## Contributing

Feel free to open issues or submit pull requests if you find any bugs or have feature requests.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
