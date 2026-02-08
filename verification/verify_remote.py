from playwright.sync_api import sync_playwright
import time

def run():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page()

        try:
            # 1. Wallet Tab
            print("Navigating to Wallet...")
            page.goto("http://localhost:3000")
            page.click("button[data-tab='wallet']")
            page.wait_for_selector("#wallet-balance")
            page.screenshot(path="verification/wallet_tab.png")

            # 2. Remote Node Selector
            print("Checking Node Selector...")
            page.wait_for_selector("#node-selector")
            page.screenshot(path="verification/node_selector.png")

            print("Screenshots captured in verification/")

        except Exception as e:
            print(f"Error: {e}")
            page.screenshot(path="verification/error_remote.png")
        finally:
            browser.close()

if __name__ == "__main__":
    run()
