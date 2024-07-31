# Chat with Local Files

With GPTScript interacting with local files is simple and powerful. This can help you streamline repetitive or data-intensive tasks. In this guide, we'll build a script that can query Excel files, CSVs, and PDFs. We'll then use the script to read, transform, and utilize the data in these files.

## Too Long; Didn't Read

:::warning
The below command will allow GPTScript to work with the files in your ~/Documents directory. Change the directory if you want to restrict it.
:::

Want to start using this script now? Just run:
```
gptscript --workspace=~/Documents github.com/gptscript-ai/local-files-demo
```

## Getting Started
The rest of this guide will walk you through building and using a data processing assistant. We'll be explaining the how, what, and why along the way.

First, let's get some sample data to work with. You can clone our repo with our sample data:
```
git clone https://github.com/gptscript-ai/local-files-demo.git
cd local-files-demo
```

Next, open up a new gptscript file in your favorite editor. We'll call the file data-assistant.gpt.
```
vim data-assistant.gpt
```
All edits below are assumed to be in this file.

### Create the Assistant
Put this in the gpt file:
```
Name: Your Data Processing Assitant
Description: An asistant to help you with processing data found in files on your workstation. Helpful for querying spreadsheets, CSVs, JSON files, and pdfs.
Tools: github.com/gptscript-ai/structured-data-querier, github.com/gptscript-ai/pdf-reader
Context: github.com/gptscript-ai/context/workspace
Chat: true

You are a helpful data processing assistant. Your goal is to help the user with data processing. Help the user accomplish their tasks using the tools you have. When the user starts this chat, just say hi, introduce yourself, and ask what you can help with.
```
This is actually the entirety of the script. We're packing a lot of power into just a handful of lines here. Let's talk through them.

**Name and Description** help the LLM understand the purpose of this tool. You should always have meaningful names and descriptions.

The **Tools: ...** stanza pulls two useful tools into this assistant.

The [structured-data-querier](https://github.com/gptscript-ai/structured-data-querier) makes it possible to query csv, xlsx, and json files as though they were SQL databases (using an application called [DuckDB](https://duckdb.org/)). This is extremely powerful when combined with the power of LLMs because it let's you ask natural language questions that the LLM can then translate to SQL.

The [pdf-reader](https://github.com/gptscript-ai/pdf-reader) isn't quite as exciting, but still useful. It parses and reads PDFs and returns the contents to the LLM. This will put the entire contents in your chat context, so it's not appropriate for extremely large PDFs, but it's handy for smaller ones.

**Context: github.com/gptscript-ai/context/workspace** introduces a context tool that makes this assistant "workspace" aware. It's description reads:
> Adds the workspace and tools needed to access the workspace to the current context

That translates to telling the LLM what the workspace directory is and instructing it to use that directory for reading and writing files. As we saw above, you can specify a workspace like this:
```
gptscript --workspace=/Your/path/here ...
```
If you don't specify one, a temporary directory will be created and used for the workspace.

This context also shares the `sys.read`, `sys.write`, and `sys.ls` built-in tools with the assistant so that it automatically has access to them.

Next we have **Chat: true**, which you've seen if you looked at any of our other guides. This makes the current tool "chat-able". We refer to chatable tools as agents or assistants.

Finally, we have the prompt:
> You are a helpful data processing assistant. Your goal is to help the user with data processing tasks. Help the user accomplish their tasks using the tools you have. When the user starts this chat, just say hi, introduce yourself, and ask what you can help with.


## Using the Assistant
Once again, that's all there is to this assistant. You can start using it by specifying your own workspace or using our sample-data directory as the workspace. Assuming you're using our sample data and have followed these instructions, here's how you launch it:
```
gptscript --workspace=./sample-data data-assistant.gpt
```

Here's a few sample interactions with these files.

### Cleaning up data
```
> whats in the key contacts file?

  The  key_contacts.txt  file contains the following information:

    Contains location, name & email
    Scranton - Michael michael@gptscript-fakeco.com
    Stamford Martin marty@gptscript-fakeco.com
    Nashua - Rachael, rach@gptscript-fakeco.com
    Utica,Ross - ross@gptscript-fakeco.com
    Albany    Allen    allen@gptscript-fakeco.com
    Akron;Pam;pam@gptscript-fakeco.com
    Camden. Kali. kali@gptscript-fakeco.com
    Yonkers - Meredith - meredith@gptscript-fakeco.com
    Rochester Kevin
    Syracuse William will@gptscript-fakeco.com

  Is there anything specific you would like to do with this information?

> That looks bad. Can you make a cleaned up csv of it?

  Sure, I'll clean up the data and create a CSV file for you. Here's the cleaned-up data in CSV format:

    Location,Name,Email
    Scranton,Michael,michael@gptscript-fakeco.com
    Stamford,Martin,marty@gptscript-fakeco.com
    Nashua,Rachael,rach@gptscript-fakeco.com
    Utica,Ross,ross@gptscript-fakeco.com
    Albany,Allen,allen@gptscript-fakeco.com
    Akron,Pam,pam@gptscript-fakeco.com
    Camden,Kali,kali@gptscript-fakeco.com
    Yonkers,Meredith,meredith@gptscript-fakeco.com
    Rochester,Kevin,
    Syracuse,William,will@gptscript-fakeco.com

  I will now create a CSV file with this cleaned-up data.
```

### Identifying and fixing data gaps
```
> is there any missing data in that csv?
...
  Yes, there is missing data in the CSV file. Specifically, the email for Kevin in Rochester is missing.

  Would you like to update this information or perform any other actions?
> yes set his email to kev@ and use the same domain as the others

  I will update Kevin's email to  kev@gptscript-fakeco.com  in the CSV file.
```

### Cross-referencing
```
> what were sales like for Kevin's location?

  To find out sales for Kevin's location, I will need to check the  yoy_sales.xlsx  file for any relevant data. Let me take a
  look at the contents of that file.
...
  The sales data for Rochester, where Kevin is located, is as follows:

    LOCATION  │ PRODUCT SKU │ MARCH 2023 SALES │ MARCH 2024 SALES │ YOY SALES GROWTH (%)
  ────────────┼─────────────┼──────────────────┼──────────────────┼───────────────────────
    Rochester │ SKU_001     │ $803.91          │ $843.10          │ 12.12%
    Rochester │ SKU_002     │ $3,013.36        │ $3,159.02        │ 13.13%
    Rochester │ SKU_003     │ $1,879.24        │ $1,968.20        │ 12.15%
    Rochester │ SKU_004     │ $4,989.81        │ $5,239.30        │ 13.88%
    Rochester │ SKU_005     │ $4,334.03        │ $4,550.73        │ 12.70%
    Rochester │ SKU_006     │ $2,143.76        │ $2,251.95        │ 13.07%
    Rochester │ SKU_007     │ $7,976.54        │ $8,375.37        │ 12.23%
    Rochester │ SKU_008     │ $313.21          │ $328.87          │ 13.80%
    Rochester │ SKU_009     │ $671.70          │ $705.29          │ 12.91%
    Rochester │ SKU_010     │ $4,084.94        │ $4,289.19        │ 13.03%

  Is there anything else you would like to know or do with this data?
```
### Pulling all the info together
```
> Let's help Kevin raise sales. What promotions do we have going on?
...
  Here are the current promotions for office paper products:

  1. Dunder Supreme Multi-Purpose Paper
    • Product SKU: SKU_001
    • Promotion Period: May 1 - May 31
    • Details: Buy two reams of Dunder Supreme and get one free. This paper is ideal for both professional and
    personal use, made from sustainably sourced forests.
  2. Scranton Elite Glossy Photo Paper
    • Product SKU: SKU_003
    • Promotion Period: May 1 - May 31
    • Details: Receive a 25% discount on every pack. This paper is designed for vibrant color reproduction and
    unrivaled photo quality, perfect for professional portfolios or family memories.
  3. MiSlin Premium Cardstock
    • Product SKU: SKU_005
    • Promotion Period: May 1 - May 31
    • Details: Buy one, get one 50% off on all cardstock purchases. This cardstock is ideal for presentations,
    report covers, and crafts, praised for its durability and smooth finish.


  These promotions can help Kevin boost sales in Rochester. Is there anything specific you would like to do with
  this information?

> craft an email summarizing performance at this location and these promotions

  Here's a draft email summarizing the performance at Kevin's location and the current promotions:

  --------

  Subject: Sales Performance and Upcoming Promotions for Rochester

  Dear Kevin,

  I hope this email finds you well.

  I wanted to take a moment to review the recent sales performance for Rochester and share some exciting promotions
  that can help boost our sales further.

  Sales Performance: The sales data for Rochester has shown promising growth across various product SKUs. Here are
  the highlights:

  • SKU_001: March 2023 Sales: $803.91 | March 2024 Sales: $843.10 | YoY Growth: 12.12%
  • SKU_002: March 2023 Sales: $3,013.36 | March 2024 Sales: $3,159.02 | YoY Growth: 13.13%
  • SKU_003: March 2023 Sales: $1,879.24 | March 2024 Sales: $1,968.20 | YoY Growth: 12.15%
  • SKU_004: March 2023 Sales: $4,989.81 | March 2024 Sales: $5,239.30 | YoY Growth: 13.88%
  • SKU_005: March 2023 Sales: $4,334.03 | March 2024 Sales: $4,550.73 | YoY Growth: 12.70%
  • SKU_006: March 2023 Sales: $2,143.76 | March 2024 Sales: $2,251.95 | YoY Growth: 13.07%
  • SKU_007: March 2023 Sales: $7,976.54 | March 2024 Sales: $8,375.37 | YoY Growth: 12.23%
  • SKU_008: March 2023 Sales: $313.21 | March 2024 Sales: $328.87 | YoY Growth: 13.80%
  • SKU_009: March 2023 Sales: $671.70 | March 2024 Sales: $705.29 | YoY Growth: 12.91%
  • SKU_010: March 2023 Sales: $4,084.94 | March 2024 Sales: $4,289.19 | YoY Growth: 13.03%

  Upcoming Promotions: To further enhance our sales, we have some exciting promotions lined up for May:

  1. Dunder Supreme Multi-Purpose Paper (SKU_001)
    • Promotion Period: May 1 - May 31
    • Details: Buy two reams and get one free. Ideal for both professional and personal use, made from sustainably
    sourced forests.
  2. Scranton Elite Glossy Photo Paper (SKU_003)
    • Promotion Period: May 1 - May 31
    • Details: Receive a 25% discount on every pack. Perfect for vibrant color reproduction and unrivaled photo
    quality.
  3. MiSlin Premium Cardstock (SKU_005)
    • Promotion Period: May 1 - May 31
    • Details: Buy one, get one 50% off on all cardstock purchases. Ideal for presentations, report covers, and
    crafts.


  These promotions are designed to attract more customers and increase our sales. Please ensure that these
  promotions are prominently displayed and communicated to our customers.

  If you have any questions or need further assistance, feel free to reach out.

  Best regards,

  [Your Name] [Your Position] [Your Contact Information]

  --------

  Feel free to customize this email further to better suit your needs. Let me know if there's anything else I can
  assist you with!
```
Try it out yourself and see what you can come up with.

## Next steps

Hopefully you've found this guide helpful. From here, you have several options:

- You can checkout out some of our other guides available in this section of the docs
- You can dive deeper into the options available when [writing script](/tools/gpt-file-reference)
