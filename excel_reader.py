import openpyxl, json

wb = openpyxl.load_workbook("spending_data.xlsx", data_only=True)
ws = wb["Transactions"]

txns = []
for row in ws.iter_rows(min_row=2, values_only=True):
    if not row[0]: continue
    txns.append({
        "date": str(row[0]),
        "merchant": str(row[1]),
        "category": str(row[2]),
        "location": str(row[3]) if row[3] else "",
        "amount": float(row[4]),
        "month": str(row[5]),
        "is_credit": bool(row[6])
    })

print(json.dumps(txns))
