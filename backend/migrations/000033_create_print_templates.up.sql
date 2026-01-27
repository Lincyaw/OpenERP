-- Migration: create_print_templates
-- Created: 2026-01-27
-- Description: Create print_templates and print_jobs tables for the printing module

-- Create print_templates table
CREATE TABLE IF NOT EXISTS print_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_type VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    paper_size VARCHAR(20) NOT NULL DEFAULT 'A4',
    orientation VARCHAR(20) NOT NULL DEFAULT 'PORTRAIT',
    margin_top INTEGER NOT NULL DEFAULT 10,
    margin_right INTEGER NOT NULL DEFAULT 10,
    margin_bottom INTEGER NOT NULL DEFAULT 10,
    margin_left INTEGER NOT NULL DEFAULT 10,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1,

    -- Constraints
    CONSTRAINT chk_print_template_document_type CHECK (document_type IN (
        'SALES_ORDER', 'SALES_DELIVERY', 'SALES_RECEIPT', 'SALES_RETURN',
        'PURCHASE_ORDER', 'PURCHASE_RECEIVING', 'PURCHASE_RETURN',
        'RECEIPT_VOUCHER', 'PAYMENT_VOUCHER', 'STOCK_TAKING'
    )),
    CONSTRAINT chk_print_template_paper_size CHECK (paper_size IN (
        'A4', 'A5', 'RECEIPT_58MM', 'RECEIPT_80MM', 'CONTINUOUS_241'
    )),
    CONSTRAINT chk_print_template_orientation CHECK (orientation IN ('PORTRAIT', 'LANDSCAPE')),
    CONSTRAINT chk_print_template_status CHECK (status IN ('ACTIVE', 'INACTIVE')),

    -- Unique constraint: only one template with same name per doc type per tenant
    CONSTRAINT uq_print_template_name_doc_type UNIQUE (tenant_id, document_type, name)
);

-- Indexes for print_templates
CREATE INDEX IF NOT EXISTS idx_print_templates_tenant_id ON print_templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_print_templates_document_type ON print_templates(document_type);
CREATE INDEX IF NOT EXISTS idx_print_templates_status ON print_templates(status);
CREATE INDEX IF NOT EXISTS idx_print_templates_is_default ON print_templates(tenant_id, document_type, is_default) WHERE is_default = TRUE;

-- Create print_jobs table
CREATE TABLE IF NOT EXISTS print_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES print_templates(id) ON DELETE RESTRICT,
    document_type VARCHAR(50) NOT NULL,
    document_id UUID NOT NULL,
    document_number VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    copies INTEGER NOT NULL DEFAULT 1,
    pdf_url TEXT,
    error_message TEXT,
    printed_at TIMESTAMP WITH TIME ZONE,
    printed_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1,

    -- Constraints
    CONSTRAINT chk_print_job_document_type CHECK (document_type IN (
        'SALES_ORDER', 'SALES_DELIVERY', 'SALES_RECEIPT', 'SALES_RETURN',
        'PURCHASE_ORDER', 'PURCHASE_RECEIVING', 'PURCHASE_RETURN',
        'RECEIPT_VOUCHER', 'PAYMENT_VOUCHER', 'STOCK_TAKING'
    )),
    CONSTRAINT chk_print_job_status CHECK (status IN ('PENDING', 'RENDERING', 'COMPLETED', 'FAILED')),
    CONSTRAINT chk_print_job_copies CHECK (copies > 0 AND copies <= 10)
);

-- Indexes for print_jobs
CREATE INDEX IF NOT EXISTS idx_print_jobs_tenant_id ON print_jobs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_template_id ON print_jobs(template_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_document ON print_jobs(tenant_id, document_type, document_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_status ON print_jobs(status);
CREATE INDEX IF NOT EXISTS idx_print_jobs_created_at ON print_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_print_jobs_printed_by ON print_jobs(printed_by);

-- Triggers for updated_at
CREATE TRIGGER update_print_templates_updated_at
    BEFORE UPDATE ON print_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_print_jobs_updated_at
    BEFORE UPDATE ON print_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE print_templates IS 'Stores print template definitions for various business documents';
COMMENT ON COLUMN print_templates.document_type IS 'Type of document this template is for (SALES_ORDER, SALES_DELIVERY, etc.)';
COMMENT ON COLUMN print_templates.content IS 'HTML template content with Go template syntax';
COMMENT ON COLUMN print_templates.paper_size IS 'Paper size for printing (A4, A5, RECEIPT_58MM, etc.)';
COMMENT ON COLUMN print_templates.is_default IS 'Whether this is the default template for the document type';

COMMENT ON TABLE print_jobs IS 'Stores print job history for audit and reprint purposes';
COMMENT ON COLUMN print_jobs.pdf_url IS 'URL/path to the generated PDF file';
COMMENT ON COLUMN print_jobs.error_message IS 'Error message if job failed';

-- ============================================================================
-- Seed default templates for the default tenant
-- ============================================================================

-- Sales Delivery Templates
INSERT INTO print_templates (tenant_id, document_type, name, description, content, paper_size, orientation, margin_top, margin_right, margin_bottom, margin_left, is_default, status)
VALUES
-- SALES_DELIVERY - A4 (Default)
('00000000-0000-0000-0000-000000000001', 'SALES_DELIVERY', '销售发货单-A4', '标准A4尺寸销售发货单/送货单模板，包含客户信息、商品明细、签收栏',
$$<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>销售发货单 - {{ .Document.DeliveryNo }}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: "Microsoft YaHei", "SimSun", Arial, sans-serif; font-size: 12px; line-height: 1.5; color: #333; }
        .page { width: 190mm; padding: 5mm; }
        .header { text-align: center; margin-bottom: 15px; border-bottom: 2px solid #333; padding-bottom: 10px; }
        .header .title { font-size: 22px; font-weight: bold; letter-spacing: 4px; }
        .header .company-name { font-size: 14px; margin-top: 5px; color: #666; }
        .info-section { display: flex; justify-content: space-between; margin-bottom: 10px; font-size: 11px; }
        .info-left, .info-right { width: 48%; }
        .info-row { display: flex; margin-bottom: 4px; }
        .info-label { width: 70px; font-weight: bold; color: #555; }
        .info-value { flex: 1; border-bottom: 1px solid #ddd; padding-left: 5px; }
        .items-table { width: 100%; border-collapse: collapse; margin-bottom: 10px; }
        .items-table th, .items-table td { border: 1px solid #333; padding: 6px 8px; text-align: center; }
        .items-table th { background-color: #f5f5f5; font-weight: bold; font-size: 11px; }
        .items-table td { font-size: 11px; }
        .items-table .col-name { text-align: left; }
        .items-table .col-qty, .items-table .col-price, .items-table .col-amount { text-align: right; }
        .summary-section { display: flex; justify-content: space-between; margin-bottom: 15px; padding: 10px; background-color: #f9f9f9; border: 1px solid #ddd; }
        .total-amount { font-size: 14px; font-weight: bold; color: #c00; }
        .amount-chinese { font-size: 11px; color: #666; margin-top: 3px; }
        .signature-section { display: flex; justify-content: space-between; margin-top: 20px; padding-top: 15px; border-top: 1px solid #ddd; }
        .signature-box { width: 30%; text-align: center; }
        .signature-label { font-size: 11px; margin-bottom: 30px; }
        .signature-line { border-bottom: 1px solid #333; margin-bottom: 5px; height: 25px; }
        .footer { margin-top: 15px; padding-top: 10px; border-top: 1px solid #ddd; font-size: 10px; color: #666; display: flex; justify-content: space-between; }
        @media print { body { -webkit-print-color-adjust: exact; print-color-adjust: exact; } .page { padding: 0; } }
    </style>
</head>
<body>
    <div class="page">
        <div class="header">
            <div class="title">销 售 发 货 单</div>
            {{ if .Company.Name }}<div class="company-name">{{ .Company.Name }}</div>{{ end }}
        </div>
        <div class="info-section">
            <div class="info-left">
                <div class="info-row"><span class="info-label">单据编号:</span><span class="info-value">{{ .Document.DeliveryNo }}</span></div>
                <div class="info-row"><span class="info-label">客户名称:</span><span class="info-value">{{ .Document.Customer.Name }}</span></div>
                <div class="info-row"><span class="info-label">联系人:</span><span class="info-value">{{ .Document.Customer.Contact }}</span></div>
                <div class="info-row"><span class="info-label">联系电话:</span><span class="info-value">{{ .Document.Customer.Phone }}</span></div>
            </div>
            <div class="info-right">
                <div class="info-row"><span class="info-label">发货日期:</span><span class="info-value">{{ .Document.ShippedAtFormatted }}</span></div>
                <div class="info-row"><span class="info-label">关联订单:</span><span class="info-value">{{ default .Document.OrderNo "-" }}</span></div>
                <div class="info-row"><span class="info-label">发货仓库:</span><span class="info-value">{{ .Document.Warehouse.Name }}</span></div>
                <div class="info-row"><span class="info-label">打印日期:</span><span class="info-value">{{ .PrintDate }}</span></div>
            </div>
        </div>
        <table class="items-table">
            <thead><tr><th>序号</th><th>商品编码</th><th>商品名称</th><th>单位</th><th>数量</th><th>单价</th><th>金额</th><th>批次号</th></tr></thead>
            <tbody>{{ range .Document.Items }}<tr><td>{{ .Index }}</td><td>{{ .ProductCode }}</td><td class="col-name">{{ .ProductName }}</td><td>{{ .Unit }}</td><td class="col-qty">{{ .QuantityFormatted }}</td><td class="col-price">{{ .UnitPriceFormatted }}</td><td class="col-amount">{{ .AmountFormatted }}</td><td>{{ default .BatchNo "-" }}</td></tr>{{ end }}</tbody>
        </table>
        <div class="summary-section">
            <div><div>商品种类: {{ .Document.ItemCount }} 种</div><div>总数量: {{ formatDecimal .Document.TotalQuantity 2 }}</div></div>
            <div style="text-align: right;"><div class="total-amount">合计金额: {{ .Document.TotalAmountFormatted }}</div><div class="amount-chinese">大写: {{ moneyToChinese .Document.TotalAmount }}</div></div>
        </div>
        <div class="signature-section">
            <div class="signature-box"><div class="signature-label">发货人</div><div class="signature-line"></div><div style="font-size:10px;color:#666">日期: ____________</div></div>
            <div class="signature-box"><div class="signature-label">收货人</div><div class="signature-line"></div><div style="font-size:10px;color:#666">日期: ____________</div></div>
            <div class="signature-box"><div class="signature-label">验收人</div><div class="signature-line"></div><div style="font-size:10px;color:#666">日期: ____________</div></div>
        </div>
        <div class="footer"><div>{{ .Company.Name }}{{ if .Company.Phone }} | 电话: {{ .Company.Phone }}{{ end }}</div><div>打印时间: {{ .PrintDateTime }}</div></div>
    </div>
</body>
</html>$$,
'A4', 'PORTRAIT', 10, 10, 10, 10, TRUE, 'ACTIVE'),

-- SALES_DELIVERY - A5
('00000000-0000-0000-0000-000000000001', 'SALES_DELIVERY', '销售发货单-A5', '紧凑A5尺寸销售发货单，适合小批量发货',
$$<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>销售发货单 - {{ .Document.DeliveryNo }}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: "Microsoft YaHei", "SimSun", Arial, sans-serif; font-size: 10px; line-height: 1.4; color: #333; }
        .page { width: 138mm; padding: 3mm; }
        .header { text-align: center; margin-bottom: 8px; border-bottom: 1px solid #333; padding-bottom: 5px; }
        .header .title { font-size: 16px; font-weight: bold; letter-spacing: 2px; }
        .info-section { display: flex; justify-content: space-between; margin-bottom: 6px; font-size: 9px; }
        .info-left, .info-right { width: 48%; }
        .info-row { display: flex; margin-bottom: 2px; }
        .info-label { width: 55px; font-weight: bold; }
        .info-value { flex: 1; border-bottom: 1px solid #ddd; padding-left: 3px; }
        .items-table { width: 100%; border-collapse: collapse; margin-bottom: 6px; }
        .items-table th, .items-table td { border: 1px solid #333; padding: 3px 4px; text-align: center; font-size: 9px; }
        .items-table th { background-color: #f5f5f5; font-weight: bold; }
        .items-table .col-name { text-align: left; }
        .items-table .col-qty, .items-table .col-price, .items-table .col-amount { text-align: right; }
        .summary-section { display: flex; justify-content: space-between; margin-bottom: 8px; padding: 5px; background-color: #f9f9f9; border: 1px solid #ddd; }
        .total-amount { font-size: 11px; font-weight: bold; color: #c00; }
        .signature-section { display: flex; justify-content: space-between; margin-top: 10px; padding-top: 8px; border-top: 1px solid #ddd; }
        .signature-box { width: 30%; text-align: center; }
        .signature-label { font-size: 9px; margin-bottom: 15px; }
        .signature-line { border-bottom: 1px solid #333; height: 15px; }
        .footer { margin-top: 8px; font-size: 8px; color: #666; text-align: center; }
        @media print { body { -webkit-print-color-adjust: exact; } .page { padding: 0; } }
    </style>
</head>
<body>
    <div class="page">
        <div class="header"><div class="title">销 售 发 货 单</div>{{ if .Company.Name }}<div style="font-size:11px;margin-top:3px;color:#666">{{ .Company.Name }}</div>{{ end }}</div>
        <div class="info-section">
            <div class="info-left">
                <div class="info-row"><span class="info-label">单据编号:</span><span class="info-value">{{ .Document.DeliveryNo }}</span></div>
                <div class="info-row"><span class="info-label">客户名称:</span><span class="info-value">{{ truncate .Document.Customer.Name 15 }}</span></div>
                <div class="info-row"><span class="info-label">联系电话:</span><span class="info-value">{{ .Document.Customer.Phone }}</span></div>
            </div>
            <div class="info-right">
                <div class="info-row"><span class="info-label">发货日期:</span><span class="info-value">{{ .Document.ShippedAtFormatted }}</span></div>
                <div class="info-row"><span class="info-label">发货仓库:</span><span class="info-value">{{ .Document.Warehouse.Name }}</span></div>
                <div class="info-row"><span class="info-label">打印日期:</span><span class="info-value">{{ .PrintDate }}</span></div>
            </div>
        </div>
        <table class="items-table">
            <thead><tr><th>序</th><th>编码</th><th>商品名称</th><th>单位</th><th>数量</th><th>单价</th><th>金额</th></tr></thead>
            <tbody>{{ range .Document.Items }}<tr><td>{{ .Index }}</td><td>{{ .ProductCode }}</td><td class="col-name">{{ truncate .ProductName 20 }}</td><td>{{ .Unit }}</td><td class="col-qty">{{ .QuantityFormatted }}</td><td class="col-price">{{ .UnitPriceFormatted }}</td><td class="col-amount">{{ .AmountFormatted }}</td></tr>{{ end }}</tbody>
        </table>
        <div class="summary-section">
            <div style="font-size:9px">品种: {{ .Document.ItemCount }} | 数量: {{ formatDecimal .Document.TotalQuantity 2 }}</div>
            <div style="text-align:right"><div class="total-amount">合计: {{ .Document.TotalAmountFormatted }}</div><div style="font-size:9px;color:#666">{{ moneyToChinese .Document.TotalAmount }}</div></div>
        </div>
        <div class="signature-section">
            <div class="signature-box"><div class="signature-label">发货人</div><div class="signature-line"></div></div>
            <div class="signature-box"><div class="signature-label">收货人</div><div class="signature-line"></div></div>
            <div class="signature-box"><div class="signature-label">验收人</div><div class="signature-line"></div></div>
        </div>
        <div class="footer">{{ .Company.Name }} | {{ .PrintDateTime }}</div>
    </div>
</body>
</html>$$,
'A5', 'PORTRAIT', 10, 10, 10, 10, FALSE, 'ACTIVE'),

-- SALES_DELIVERY - Continuous 241mm
('00000000-0000-0000-0000-000000000001', 'SALES_DELIVERY', '销售发货单-连续纸', '241mm连续纸格式，适用于针式打印机多联打印',
$$<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>销售发货单 - {{ .Document.DeliveryNo }}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: "SimSun", monospace; font-size: 12px; line-height: 1.3; color: #000; }
        .page { width: 231mm; padding: 2mm; }
        .header { text-align: center; margin-bottom: 8px; border-bottom: 2px double #000; padding-bottom: 5px; }
        .header .title { font-size: 18px; font-weight: bold; letter-spacing: 3px; }
        .header .doc-no { font-size: 11px; margin-top: 3px; }
        .info-grid { display: table; width: 100%; margin-bottom: 5px; font-size: 11px; }
        .info-row { display: table-row; }
        .info-cell { display: table-cell; padding: 2px 5px; border-bottom: 1px dotted #999; }
        .info-label { width: 60px; font-weight: bold; }
        .items-table { width: 100%; border-collapse: collapse; margin-bottom: 5px; }
        .items-table th, .items-table td { border: 1px solid #000; padding: 3px 5px; text-align: center; font-size: 11px; }
        .items-table th { background-color: #eee; font-weight: bold; }
        .items-table .col-name { text-align: left; }
        .items-table .col-qty, .items-table .col-price, .items-table .col-amount { text-align: right; }
        .summary-row { display: flex; justify-content: space-between; padding: 5px; border: 1px solid #000; margin-bottom: 5px; font-size: 12px; }
        .total-amount { font-weight: bold; }
        .signature-section { display: flex; justify-content: space-between; margin-top: 8px; border-top: 1px solid #000; padding-top: 8px; }
        .signature-box { width: 22%; text-align: center; }
        .signature-label { font-size: 11px; margin-bottom: 15px; font-weight: bold; }
        .signature-line { border-bottom: 1px solid #000; height: 18px; }
        .copy-indicator { text-align: right; font-size: 10px; font-weight: bold; margin-bottom: 3px; }
        .footer { margin-top: 5px; font-size: 10px; text-align: center; border-top: 1px dotted #999; padding-top: 3px; }
    </style>
</head>
<body>
    <div class="page">
        <div class="copy-indicator">第一联: 存根联</div>
        <div class="header"><div class="title">销 售 发 货 单</div><div class="doc-no">单号: {{ .Document.DeliveryNo }} | {{ .Document.ShippedAtFormatted }}</div></div>
        <div class="info-grid">
            <div class="info-row"><span class="info-cell info-label">客户:</span><span class="info-cell">{{ .Document.Customer.Name }}</span><span class="info-cell info-label">电话:</span><span class="info-cell">{{ .Document.Customer.Phone }}</span><span class="info-cell info-label">仓库:</span><span class="info-cell">{{ .Document.Warehouse.Name }}</span></div>
            <div class="info-row"><span class="info-cell info-label">联系人:</span><span class="info-cell">{{ .Document.Customer.Contact }}</span><span class="info-cell info-label">订单号:</span><span class="info-cell">{{ default .Document.OrderNo "-" }}</span><span class="info-cell info-label">打印:</span><span class="info-cell">{{ .PrintDate }}</span></div>
        </div>
        <table class="items-table">
            <thead><tr><th>序</th><th>商品编码</th><th>商品名称</th><th>单位</th><th>数量</th><th>单价</th><th>金额</th><th>批次</th></tr></thead>
            <tbody>{{ range .Document.Items }}<tr><td>{{ .Index }}</td><td>{{ .ProductCode }}</td><td class="col-name">{{ truncate .ProductName 25 }}</td><td>{{ .Unit }}</td><td class="col-qty">{{ .QuantityFormatted }}</td><td class="col-price">{{ .UnitPriceFormatted }}</td><td class="col-amount">{{ .AmountFormatted }}</td><td>{{ default .BatchNo "-" }}</td></tr>{{ end }}</tbody>
        </table>
        <div class="summary-row"><span>品种数: {{ .Document.ItemCount }} | 总数量: {{ formatDecimal .Document.TotalQuantity 2 }}</span><span class="total-amount">合计金额: {{ .Document.TotalAmountFormatted }}</span><span>大写: {{ moneyToChinese .Document.TotalAmount }}</span></div>
        <div class="signature-section">
            <div class="signature-box"><div class="signature-label">制单人</div><div class="signature-line"></div></div>
            <div class="signature-box"><div class="signature-label">发货人</div><div class="signature-line"></div></div>
            <div class="signature-box"><div class="signature-label">收货人</div><div class="signature-line"></div></div>
            <div class="signature-box"><div class="signature-label">验收人</div><div class="signature-line"></div></div>
        </div>
        <div class="footer">{{ .Company.Name }}{{ if .Company.Phone }} | {{ .Company.Phone }}{{ end }}</div>
    </div>
</body>
</html>$$,
'CONTINUOUS_241', 'PORTRAIT', 5, 5, 5, 5, FALSE, 'ACTIVE');

-- Sales Receipt Templates
INSERT INTO print_templates (tenant_id, document_type, name, description, content, paper_size, orientation, margin_top, margin_right, margin_bottom, margin_left, is_default, status)
VALUES
-- SALES_RECEIPT - 58mm (Default)
('00000000-0000-0000-0000-000000000001', 'SALES_RECEIPT', '销售收据-58mm', '58mm热敏小票，适用于收银台热敏打印机',
$$<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>销售收据 - {{ .Document.ReceiptNo }}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: "Microsoft YaHei", "SimSun", Arial, sans-serif; font-size: 9px; line-height: 1.3; color: #000; width: 54mm; }
        .receipt { width: 54mm; padding: 1mm; }
        .header { text-align: center; padding-bottom: 3px; border-bottom: 1px dashed #000; margin-bottom: 3px; }
        .store-name { font-size: 12px; font-weight: bold; }
        .store-info { font-size: 7px; color: #333; margin-top: 2px; }
        .receipt-info { font-size: 8px; margin-bottom: 3px; padding-bottom: 3px; border-bottom: 1px dashed #000; }
        .receipt-row { display: flex; justify-content: space-between; }
        .items { margin-bottom: 3px; padding-bottom: 3px; border-bottom: 1px dashed #000; }
        .item-header { display: flex; justify-content: space-between; font-size: 8px; font-weight: bold; border-bottom: 1px solid #ccc; padding-bottom: 2px; margin-bottom: 2px; }
        .item-line { font-size: 8px; margin-bottom: 2px; }
        .item-detail { display: flex; justify-content: space-between; color: #333; padding-left: 5px; }
        .totals { margin-bottom: 3px; padding-bottom: 3px; border-bottom: 1px dashed #000; }
        .total-row { display: flex; justify-content: space-between; font-size: 8px; }
        .total-row.grand { font-size: 11px; font-weight: bold; margin-top: 2px; padding-top: 2px; border-top: 1px solid #ccc; }
        .payments { margin-bottom: 3px; font-size: 8px; }
        .payment-row { display: flex; justify-content: space-between; }
        .footer { text-align: center; font-size: 7px; color: #666; padding-top: 3px; }
        .footer .thanks { font-size: 9px; font-weight: bold; margin-bottom: 2px; }
    </style>
</head>
<body>
    <div class="receipt">
        <div class="header"><div class="store-name">{{ .Document.Store.Name }}</div>{{ if .Document.Store.Address }}<div class="store-info">{{ truncate .Document.Store.Address 30 }}</div>{{ end }}{{ if .Document.Store.Phone }}<div class="store-info">电话: {{ .Document.Store.Phone }}</div>{{ end }}</div>
        <div class="receipt-info"><div class="receipt-row"><span>单号: {{ .Document.ReceiptNo }}</span></div><div class="receipt-row"><span>时间: {{ .Document.TransactedAtFormatted }}</span></div><div class="receipt-row"><span>收银员: {{ .Document.Cashier }}</span></div></div>
        <div class="items"><div class="item-header"><span>品名</span><span>金额</span></div>{{ range .Document.Items }}<div class="item-line"><span>{{ truncate .ProductName 18 }}</span><div class="item-detail"><span>{{ formatDecimal .Quantity 0 }} x {{ formatMoney .UnitPrice }}</span><span>{{ formatMoney .Amount }}</span></div></div>{{ end }}</div>
        <div class="totals"><div class="total-row"><span>小计 ({{ .Document.ItemCount }}件)</span><span>{{ .Document.SubtotalFormatted }}</span></div>{{ if gt .Document.DiscountTotal.IntPart 0 }}<div class="total-row" style="color:#c00"><span>优惠</span><span>-{{ .Document.DiscountTotalFormatted }}</span></div>{{ end }}<div class="total-row grand"><span>合计</span><span>{{ .Document.GrandTotalFormatted }}</span></div></div>
        <div class="payments">{{ range .Document.Payments }}<div class="payment-row"><span>{{ .MethodText }}</span><span>{{ .AmountFormatted }}</span></div>{{ end }}{{ if gt .Document.Change.IntPart 0 }}<div class="payment-row" style="font-weight:bold"><span>找零</span><span>{{ .Document.ChangeFormatted }}</span></div>{{ end }}</div>
        <div class="footer"><div class="thanks">谢谢惠顾!</div><div>{{ .PrintDateTime }}</div></div>
    </div>
</body>
</html>$$,
'RECEIPT_58MM', 'PORTRAIT', 2, 2, 2, 2, TRUE, 'ACTIVE'),

-- SALES_RECEIPT - 80mm
('00000000-0000-0000-0000-000000000001', 'SALES_RECEIPT', '销售收据-80mm', '80mm热敏小票，内容更详细，适用于大型收银台',
$$<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>销售收据 - {{ .Document.ReceiptNo }}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: "Microsoft YaHei", "SimSun", Arial, sans-serif; font-size: 10px; line-height: 1.4; color: #000; width: 76mm; }
        .receipt { width: 76mm; padding: 2mm; }
        .header { text-align: center; padding-bottom: 5px; border-bottom: 2px solid #000; margin-bottom: 5px; }
        .store-name { font-size: 14px; font-weight: bold; }
        .store-info { font-size: 9px; color: #333; margin-top: 2px; }
        .receipt-info { font-size: 9px; margin-bottom: 5px; padding-bottom: 5px; border-bottom: 1px dashed #000; }
        .receipt-row { display: flex; justify-content: space-between; margin-bottom: 1px; }
        .items-table { width: 100%; border-collapse: collapse; margin-bottom: 5px; }
        .items-table th, .items-table td { padding: 3px 2px; text-align: left; font-size: 9px; }
        .items-table th { border-bottom: 1px solid #000; font-weight: bold; }
        .items-table td { border-bottom: 1px dotted #ccc; }
        .items-table .col-qty { text-align: center; }
        .items-table .col-price, .items-table .col-amount { text-align: right; }
        .totals { margin-bottom: 5px; padding: 5px 0; border-top: 1px dashed #000; border-bottom: 1px dashed #000; }
        .total-row { display: flex; justify-content: space-between; font-size: 10px; margin-bottom: 2px; }
        .total-row.grand { font-size: 13px; font-weight: bold; margin-top: 3px; padding-top: 3px; border-top: 1px solid #ccc; }
        .payments { margin-bottom: 5px; padding-bottom: 5px; border-bottom: 1px dashed #000; }
        .payments-title { font-size: 9px; font-weight: bold; margin-bottom: 3px; }
        .payment-row { display: flex; justify-content: space-between; font-size: 10px; margin-bottom: 1px; }
        .chinese-amount { text-align: center; font-size: 9px; color: #666; margin-bottom: 5px; }
        .footer { text-align: center; font-size: 8px; color: #666; padding-top: 5px; }
        .footer .thanks { font-size: 11px; font-weight: bold; margin-bottom: 3px; }
    </style>
</head>
<body>
    <div class="receipt">
        <div class="header"><div class="store-name">{{ .Document.Store.Name }}</div>{{ if .Document.Store.Address }}<div class="store-info">{{ .Document.Store.Address }}</div>{{ end }}{{ if .Document.Store.Phone }}<div class="store-info">电话: {{ .Document.Store.Phone }}</div>{{ end }}</div>
        <div class="receipt-info"><div class="receipt-row"><span>单号: {{ .Document.ReceiptNo }}</span><span>收银员: {{ .Document.Cashier }}</span></div><div class="receipt-row"><span>日期: {{ .Document.TransactedAtFormatted }}</span></div></div>
        <table class="items-table"><thead><tr><th>商品名称</th><th class="col-qty">数量</th><th class="col-price">单价</th><th class="col-amount">金额</th></tr></thead><tbody>{{ range .Document.Items }}<tr><td>{{ truncate .ProductName 14 }}</td><td class="col-qty">{{ formatDecimal .Quantity 0 }}</td><td class="col-price">{{ formatMoney .UnitPrice }}</td><td class="col-amount">{{ formatMoney .Amount }}</td></tr>{{ end }}</tbody></table>
        <div class="totals"><div class="total-row"><span>小计 ({{ .Document.ItemCount }}件 / {{ formatDecimal .Document.TotalQuantity 0 }}个)</span><span>{{ .Document.SubtotalFormatted }}</span></div>{{ if gt .Document.DiscountTotal.IntPart 0 }}<div class="total-row" style="color:#c00"><span>优惠金额</span><span>-{{ .Document.DiscountTotalFormatted }}</span></div>{{ end }}<div class="total-row grand"><span>应付金额</span><span>{{ .Document.GrandTotalFormatted }}</span></div></div>
        <div class="payments"><div class="payments-title">支付方式</div>{{ range .Document.Payments }}<div class="payment-row"><span>{{ .MethodText }}</span><span>{{ .AmountFormatted }}</span></div>{{ end }}{{ if gt .Document.Change.IntPart 0 }}<div class="payment-row" style="font-weight:bold;font-size:11px"><span>找零</span><span>{{ .Document.ChangeFormatted }}</span></div>{{ end }}</div>
        <div class="chinese-amount">金额大写: {{ moneyToChinese .Document.GrandTotal }}</div>
        <div class="footer"><div class="thanks">谢谢惠顾，欢迎再来!</div><div>打印时间: {{ .PrintDateTime }}</div></div>
    </div>
</body>
</html>$$,
'RECEIPT_80MM', 'PORTRAIT', 2, 2, 2, 2, FALSE, 'ACTIVE'),

-- SALES_RECEIPT - A5
('00000000-0000-0000-0000-000000000001', 'SALES_RECEIPT', '销售收据-A5', 'A5尺寸销售收据，正式发票替代品',
$$<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>销售收据 - {{ .Document.ReceiptNo }}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: "Microsoft YaHei", "SimSun", Arial, sans-serif; font-size: 11px; line-height: 1.5; color: #333; }
        .page { width: 138mm; padding: 5mm; }
        .header { text-align: center; margin-bottom: 10px; border-bottom: 2px solid #333; padding-bottom: 8px; }
        .header .title { font-size: 18px; font-weight: bold; letter-spacing: 3px; }
        .header .store-name { font-size: 12px; margin-top: 5px; color: #666; }
        .info-section { display: flex; justify-content: space-between; margin-bottom: 10px; padding: 8px; background-color: #f9f9f9; border: 1px solid #ddd; }
        .info-row { margin-bottom: 3px; font-size: 10px; }
        .info-label { font-weight: bold; display: inline-block; width: 50px; }
        .items-table { width: 100%; border-collapse: collapse; margin-bottom: 10px; }
        .items-table th, .items-table td { border: 1px solid #333; padding: 5px 8px; text-align: center; font-size: 10px; }
        .items-table th { background-color: #f0f0f0; font-weight: bold; }
        .items-table .col-name { text-align: left; }
        .items-table .col-qty, .items-table .col-price, .items-table .col-discount, .items-table .col-amount { text-align: right; }
        .summary-section { display: flex; justify-content: space-between; margin-bottom: 10px; }
        .summary-row { font-size: 10px; margin-bottom: 3px; }
        .total-row { font-size: 14px; font-weight: bold; margin-top: 5px; padding-top: 5px; border-top: 1px solid #ddd; }
        .amount-chinese { font-size: 10px; color: #666; }
        .payments-section { padding: 8px; background-color: #f5f5f5; border: 1px solid #ddd; margin-bottom: 10px; }
        .payments-title { font-size: 11px; font-weight: bold; margin-bottom: 5px; }
        .payment-row { display: flex; justify-content: space-between; font-size: 10px; margin-bottom: 2px; }
        .footer { text-align: center; padding-top: 10px; border-top: 1px dashed #999; }
        .footer .thanks { font-size: 12px; font-weight: bold; margin-bottom: 5px; }
        .footer .info { font-size: 9px; color: #666; }
        @media print { body { -webkit-print-color-adjust: exact; } .page { padding: 0; } }
    </style>
</head>
<body>
    <div class="page">
        <div class="header"><div class="title">销 售 收 据</div><div class="store-name">{{ .Document.Store.Name }}</div>{{ if .Document.Store.Address }}<div style="font-size:10px;color:#888;margin-top:3px">{{ .Document.Store.Address }}</div>{{ end }}</div>
        <div class="info-section"><div><div class="info-row"><span class="info-label">单号:</span>{{ .Document.ReceiptNo }}</div><div class="info-row"><span class="info-label">日期:</span>{{ .Document.TransactedAtFormatted }}</div></div><div><div class="info-row"><span class="info-label">收银员:</span>{{ .Document.Cashier }}</div><div class="info-row"><span class="info-label">打印:</span>{{ .PrintDateTime }}</div></div></div>
        <table class="items-table"><thead><tr><th>序号</th><th>商品名称</th><th>数量</th><th>单价</th><th>折扣</th><th>金额</th></tr></thead><tbody>{{ range .Document.Items }}<tr><td>{{ .Index }}</td><td class="col-name">{{ .ProductName }}</td><td class="col-qty">{{ formatDecimal .Quantity 0 }}</td><td class="col-price">{{ formatMoney .UnitPrice }}</td><td class="col-discount">{{ if gt .Discount.IntPart 0 }}-{{ formatMoney .Discount }}{{ else }}-{{ end }}</td><td class="col-amount">{{ formatMoney .Amount }}</td></tr>{{ end }}</tbody></table>
        <div class="summary-section"><div style="width:50%"><div class="summary-row">商品种类: {{ .Document.ItemCount }}</div><div class="summary-row">商品数量: {{ formatDecimal .Document.TotalQuantity 0 }}</div></div><div style="width:45%;text-align:right"><div class="summary-row">商品小计: {{ .Document.SubtotalFormatted }}</div>{{ if gt .Document.DiscountTotal.IntPart 0 }}<div class="summary-row" style="color:#c00">优惠金额: -{{ .Document.DiscountTotalFormatted }}</div>{{ end }}<div class="total-row">应付金额: {{ .Document.GrandTotalFormatted }}</div><div class="amount-chinese">大写: {{ moneyToChinese .Document.GrandTotal }}</div></div></div>
        <div class="payments-section"><div class="payments-title">支付方式</div>{{ range .Document.Payments }}<div class="payment-row"><span>{{ .MethodText }}</span><span>{{ .AmountFormatted }}</span></div>{{ end }}{{ if gt .Document.Change.IntPart 0 }}<div class="payment-row" style="font-weight:bold;color:#060"><span>找零</span><span>{{ .Document.ChangeFormatted }}</span></div>{{ end }}</div>
        <div class="footer"><div class="thanks">谢谢惠顾!</div>{{ if .Document.Store.TaxID }}<div class="info">税号: {{ .Document.Store.TaxID }}</div>{{ end }}<div class="info">请妥善保管本收据，如有问题请在7天内持本收据联系我们</div></div>
    </div>
</body>
</html>$$,
'A5', 'PORTRAIT', 10, 10, 10, 10, FALSE, 'ACTIVE');

-- Purchase Receiving Templates
INSERT INTO print_templates (tenant_id, document_type, name, description, content, paper_size, orientation, margin_top, margin_right, margin_bottom, margin_left, is_default, status)
VALUES
-- PURCHASE_RECEIVING - A4 (Default)
('00000000-0000-0000-0000-000000000001', 'PURCHASE_RECEIVING', '采购入库单-A4', '标准A4尺寸采购入库单，包含供应商信息、商品明细、批次、质检状态',
$$<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>采购入库单 - {{ .Document.ReceivingNo }}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: "Microsoft YaHei", "SimSun", Arial, sans-serif; font-size: 12px; line-height: 1.5; color: #333; }
        .page { width: 190mm; padding: 5mm; }
        .header { text-align: center; margin-bottom: 15px; border-bottom: 2px solid #333; padding-bottom: 10px; }
        .header .title { font-size: 22px; font-weight: bold; letter-spacing: 4px; }
        .header .company-name { font-size: 14px; margin-top: 5px; color: #666; }
        .info-section { display: flex; justify-content: space-between; margin-bottom: 10px; font-size: 11px; }
        .info-left, .info-right { width: 48%; }
        .info-row { display: flex; margin-bottom: 4px; }
        .info-label { width: 70px; font-weight: bold; color: #555; }
        .info-value { flex: 1; border-bottom: 1px solid #ddd; padding-left: 5px; }
        .supplier-box { margin-bottom: 10px; padding: 8px; border: 1px solid #ccc; background-color: #f9f9f9; }
        .supplier-title { font-size: 11px; font-weight: bold; margin-bottom: 5px; color: #555; }
        .supplier-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 3px; font-size: 11px; }
        .items-table { width: 100%; border-collapse: collapse; margin-bottom: 10px; }
        .items-table th, .items-table td { border: 1px solid #333; padding: 6px 8px; text-align: center; font-size: 11px; }
        .items-table th { background-color: #f5f5f5; font-weight: bold; }
        .items-table .col-name { text-align: left; }
        .items-table .col-ordered, .items-table .col-received, .items-table .col-price, .items-table .col-amount { text-align: right; }
        .status-pass { color: #060; font-weight: bold; }
        .status-reject { color: #c00; font-weight: bold; }
        .status-pending { color: #f90; }
        .summary-section { display: flex; justify-content: space-between; margin-bottom: 15px; padding: 10px; background-color: #f9f9f9; border: 1px solid #ddd; }
        .total-amount { font-size: 14px; font-weight: bold; color: #c00; }
        .amount-chinese { font-size: 11px; color: #666; margin-top: 3px; }
        .signature-section { display: flex; justify-content: space-between; margin-top: 20px; padding-top: 15px; border-top: 1px solid #ddd; }
        .signature-box { width: 23%; text-align: center; }
        .signature-label { font-size: 11px; margin-bottom: 30px; }
        .signature-line { border-bottom: 1px solid #333; margin-bottom: 5px; height: 25px; }
        .footer { margin-top: 15px; padding-top: 10px; border-top: 1px solid #ddd; font-size: 10px; color: #666; display: flex; justify-content: space-between; }
        @media print { body { -webkit-print-color-adjust: exact; } .page { padding: 0; } }
    </style>
</head>
<body>
    <div class="page">
        <div class="header"><div class="title">采 购 入 库 单</div>{{ if .Company.Name }}<div class="company-name">{{ .Company.Name }}</div>{{ end }}</div>
        <div class="info-section">
            <div class="info-left">
                <div class="info-row"><span class="info-label">入库单号:</span><span class="info-value">{{ .Document.ReceivingNo }}</span></div>
                <div class="info-row"><span class="info-label">采购单号:</span><span class="info-value">{{ default .Document.PurchaseOrderNo "-" }}</span></div>
                <div class="info-row"><span class="info-label">入库仓库:</span><span class="info-value">{{ .Document.Warehouse.Name }}</span></div>
            </div>
            <div class="info-right">
                <div class="info-row"><span class="info-label">入库日期:</span><span class="info-value">{{ .Document.ReceivedAtFormatted }}</span></div>
                <div class="info-row"><span class="info-label">收货人:</span><span class="info-value">{{ .Document.ReceivedBy }}</span></div>
                <div class="info-row"><span class="info-label">打印日期:</span><span class="info-value">{{ .PrintDate }}</span></div>
            </div>
        </div>
        <div class="supplier-box"><div class="supplier-title">供应商信息</div><div class="supplier-grid"><div><strong>名称:</strong> {{ .Document.Supplier.Name }}</div><div><strong>联系人:</strong> {{ .Document.Supplier.Contact }}</div><div><strong>电话:</strong> {{ .Document.Supplier.Phone }}</div><div><strong>地址:</strong> {{ default .Document.Supplier.Address "-" }}</div></div></div>
        <table class="items-table">
            <thead><tr><th>序号</th><th>商品编码</th><th>商品名称</th><th>单位</th><th>订购量</th><th>入库量</th><th>单价</th><th>金额</th><th>批次号</th><th>有效期</th><th>质检</th></tr></thead>
            <tbody>{{ range .Document.Items }}<tr><td>{{ .Index }}</td><td>{{ .ProductCode }}</td><td class="col-name">{{ .ProductName }}</td><td>{{ .Unit }}</td><td class="col-ordered">{{ .OrderedQuantityFormatted }}</td><td class="col-received">{{ .ReceivedQuantityFormatted }}</td><td class="col-price">{{ .UnitPriceFormatted }}</td><td class="col-amount">{{ .AmountFormatted }}</td><td>{{ default .BatchNo "-" }}</td><td>{{ default .ExpiryDateFormatted "-" }}</td><td>{{ if eq .QualityStatus "PASS" }}<span class="status-pass">合格</span>{{ else if eq .QualityStatus "REJECT" }}<span class="status-reject">不合格</span>{{ else }}<span class="status-pending">待检</span>{{ end }}</td></tr>{{ end }}</tbody>
        </table>
        <div class="summary-section"><div><div>商品种类: {{ .Document.ItemCount }} 种</div><div>入库总数: {{ formatDecimal .Document.TotalQuantity 2 }}</div></div><div style="text-align:right"><div class="total-amount">合计金额: {{ .Document.TotalAmountFormatted }}</div><div class="amount-chinese">大写: {{ moneyToChinese .Document.TotalAmount }}</div></div></div>
        <div class="signature-section">
            <div class="signature-box"><div class="signature-label">制单人</div><div class="signature-line"></div><div style="font-size:10px;color:#666">日期: ____________</div></div>
            <div class="signature-box"><div class="signature-label">收货人</div><div class="signature-line">{{ if .Document.ReceivedBy }}{{ .Document.ReceivedBy }}{{ end }}</div><div style="font-size:10px;color:#666">日期: {{ .Document.ReceivedAtFormatted }}</div></div>
            <div class="signature-box"><div class="signature-label">质检员</div><div class="signature-line">{{ if .Document.InspectedBy }}{{ .Document.InspectedBy }}{{ end }}</div><div style="font-size:10px;color:#666">日期: ____________</div></div>
            <div class="signature-box"><div class="signature-label">仓库主管</div><div class="signature-line"></div><div style="font-size:10px;color:#666">日期: ____________</div></div>
        </div>
        <div class="footer"><div>{{ .Company.Name }}{{ if .Company.Phone }} | 电话: {{ .Company.Phone }}{{ end }}</div><div>打印时间: {{ .PrintDateTime }}</div></div>
    </div>
</body>
</html>$$,
'A4', 'PORTRAIT', 10, 10, 10, 10, TRUE, 'ACTIVE'),

-- PURCHASE_RECEIVING - Continuous 241mm
('00000000-0000-0000-0000-000000000001', 'PURCHASE_RECEIVING', '采购入库单-连续纸', '241mm连续纸格式，适用于仓库针式打印机',
$$<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>采购入库单 - {{ .Document.ReceivingNo }}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: "SimSun", monospace; font-size: 12px; line-height: 1.3; color: #000; }
        .page { width: 231mm; padding: 2mm; }
        .header { text-align: center; margin-bottom: 8px; border-bottom: 2px double #000; padding-bottom: 5px; }
        .header .title { font-size: 18px; font-weight: bold; letter-spacing: 3px; }
        .header .doc-no { font-size: 11px; margin-top: 3px; }
        .info-grid { display: table; width: 100%; margin-bottom: 5px; font-size: 11px; }
        .info-row { display: table-row; }
        .info-cell { display: table-cell; padding: 2px 5px; border-bottom: 1px dotted #999; }
        .info-label { width: 55px; font-weight: bold; }
        .items-table { width: 100%; border-collapse: collapse; margin-bottom: 5px; }
        .items-table th, .items-table td { border: 1px solid #000; padding: 3px 5px; text-align: center; font-size: 11px; }
        .items-table th { background-color: #eee; font-weight: bold; }
        .items-table .col-name { text-align: left; }
        .items-table .col-ordered, .items-table .col-received, .items-table .col-price, .items-table .col-amount { text-align: right; }
        .summary-row { display: flex; justify-content: space-between; padding: 5px; border: 1px solid #000; margin-bottom: 5px; font-size: 12px; }
        .total-amount { font-weight: bold; }
        .signature-section { display: flex; justify-content: space-between; margin-top: 8px; border-top: 1px solid #000; padding-top: 8px; }
        .signature-box { width: 23%; text-align: center; }
        .signature-label { font-size: 11px; margin-bottom: 15px; font-weight: bold; }
        .signature-line { border-bottom: 1px solid #000; height: 18px; }
        .copy-indicator { text-align: right; font-size: 10px; font-weight: bold; margin-bottom: 3px; }
        .footer { margin-top: 5px; font-size: 10px; text-align: center; border-top: 1px dotted #999; padding-top: 3px; }
    </style>
</head>
<body>
    <div class="page">
        <div class="copy-indicator">第一联: 财务联</div>
        <div class="header"><div class="title">采 购 入 库 单</div><div class="doc-no">单号: {{ .Document.ReceivingNo }} | {{ .Document.ReceivedAtFormatted }}</div></div>
        <div class="info-grid">
            <div class="info-row"><span class="info-cell info-label">供应商:</span><span class="info-cell">{{ .Document.Supplier.Name }}</span><span class="info-cell info-label">联系人:</span><span class="info-cell">{{ .Document.Supplier.Contact }}</span><span class="info-cell info-label">电话:</span><span class="info-cell">{{ .Document.Supplier.Phone }}</span></div>
            <div class="info-row"><span class="info-cell info-label">采购单:</span><span class="info-cell">{{ default .Document.PurchaseOrderNo "-" }}</span><span class="info-cell info-label">仓库:</span><span class="info-cell">{{ .Document.Warehouse.Name }}</span><span class="info-cell info-label">收货人:</span><span class="info-cell">{{ .Document.ReceivedBy }}</span></div>
        </div>
        <table class="items-table">
            <thead><tr><th>序</th><th>商品编码</th><th>商品名称</th><th>单位</th><th>订购量</th><th>入库量</th><th>单价</th><th>金额</th><th>批次号</th><th>有效期</th></tr></thead>
            <tbody>{{ range .Document.Items }}<tr><td>{{ .Index }}</td><td>{{ .ProductCode }}</td><td class="col-name">{{ truncate .ProductName 25 }}</td><td>{{ .Unit }}</td><td class="col-ordered">{{ .OrderedQuantityFormatted }}</td><td class="col-received">{{ .ReceivedQuantityFormatted }}</td><td class="col-price">{{ .UnitPriceFormatted }}</td><td class="col-amount">{{ .AmountFormatted }}</td><td>{{ default .BatchNo "-" }}</td><td>{{ default .ExpiryDateFormatted "-" }}</td></tr>{{ end }}</tbody>
        </table>
        <div class="summary-row"><span>品种数: {{ .Document.ItemCount }} | 入库总数: {{ formatDecimal .Document.TotalQuantity 2 }}</span><span class="total-amount">合计金额: {{ .Document.TotalAmountFormatted }}</span><span>大写: {{ moneyToChinese .Document.TotalAmount }}</span></div>
        <div class="signature-section">
            <div class="signature-box"><div class="signature-label">制单人</div><div class="signature-line"></div></div>
            <div class="signature-box"><div class="signature-label">收货人</div><div class="signature-line"></div></div>
            <div class="signature-box"><div class="signature-label">质检员</div><div class="signature-line"></div></div>
            <div class="signature-box"><div class="signature-label">仓库主管</div><div class="signature-line"></div></div>
        </div>
        <div class="footer">{{ .Company.Name }}{{ if .Company.Phone }} | {{ .Company.Phone }}{{ end }} | 打印: {{ .PrintDateTime }}</div>
    </div>
</body>
</html>$$,
'CONTINUOUS_241', 'PORTRAIT', 5, 5, 5, 5, FALSE, 'ACTIVE');
