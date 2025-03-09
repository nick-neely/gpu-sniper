# GPU Sniper

## Overview

GPU Sniper is a lightweight, high-speed Go script designed to automatically monitor Amazon's website for GPU availability. Originally conceived as a minimal personal-use tool, it checks for a specific GPU model, logs stock status, and lays the groundwork for future extensions.

## Purpose

Due to persistent GPU shortages and rapid price fluctuations, manual monitoring is inefficient. GPU Sniper automates the process, increasing your chance to secure a highly desired GPU, currently focusing on Amazon as the target retailer.

## Why It's Here

Inspired by the need for a fast and efficient GPU purchase bot, GPU Sniper was developed to:

- Monitor GPU stock status on Amazon.
- Log checks with detailed timestamps and color-coded outputs.
- Provide a basis for future expansion to support multiple retailers and more advanced features.

## Anti-Bot Measures

GPU Sniper employs multiple strategies to appear more like genuine human activity:

- **Random User Agents:** Rotates through various realistic browser user agent strings for every HTTP request.
- **Jittered Timing:** Introduces random delays (jitter) to polling intervals, ensuring requests are less predictable.
- **Visiting Related Pages:** Occasionally accesses Amazon's related pages (e.g., PC Components, deals) to simulate natural browsing behavior.
- **Exponential Backoff:** Applies exponential backoff when encountering errors or rate limits, preventing aggressive retry loops.
- **Cookie Management:** Maintains and reuses cookies across requests, mimicking a consistent browser session.
- **Adaptive Polling Frequency:** Adjusts the frequency of checks based on the time of day (e.g., fewer checks during late night hours, slightly increased intervals during peak traffic periods).

## How to Use

### Before Running the Script

1. Set your Amazon Product ID in `config/config.go`.
   - Find your product's ID (ASIN) in the URL of the product page on Amazon. For example, in `https://www.amazon.com/gp/B0DVCH9WJH`, the product ID is `B0DVCH9WJH`.

### Running the Script

1. Install Go from [the official Go installation page](https://go.dev/doc/install).
2. Clone the repository:

   ```bash
   git clone https://github.com/nick-neely/gpu-sniper.git
   ```

3. Navigate to the project directory:

   ```bash
   cd gpu-sniper
   ```

4. Run the script:

   ```bash
   go run main.go
   ```

### Logging and Terminal Output

- Informational messages, success logs, error logs, and warnings are color-coded.
- A header displays the active target GPU, retailer URL, and anti-bot measures.
- A periodic status update shows the number of checks performed, the current interval, and the time since the last check.

## Add-to-Cart Automation

GPU Sniper automatically generates the add-to-cart link for the product by combining the product identifier with Amazon's URL pattern. Once the product is detected in stock, the script auto-clicks this link, opening it in your default browser. Note that this action serves as an alert mechanism and does not automatically complete the purchase.

## Configuration Options

Configuration is managed via the `config/config.go` file:

- **ProductID & Retailer URL**:
  - The `ProductID` defines which product is followed on Amazon.
  - The `RetailerURL` is constructed based on the ProductID.
- **GPU Target**:
  - The script is currently hard-coded to monitor the "NVIDIA RTX 5090". This can be updated as needed.
- **Polling Interval**:
  - The default polling interval for checks is configurable (`DefaultPollingInterval` in `config/config.go`).
- **Retry Configurations**:
  - The script includes retry settings (`DefaultRetryConfig`, `StockCheckRetryConfig`, `RelatedPageRetryConfig`) that manage retry logic for HTTP requests.

## Future Enhancements

- Adding support for multiple GPU models.
- Integrating additional retailers beyond Amazon.
- Expanding configuration options through command-line flags or a configuration file.
- Implementing automatic purchase functionality with configurable price thresholds and purchase criteria.

---

Happy Sniping!
