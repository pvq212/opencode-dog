import { useState } from 'react';
import { Title } from 'react-admin';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import Typography from '@mui/material/Typography';
import Alert from '@mui/material/Alert';
import GitLabIcon from '@mui/icons-material/Code';
import SlackIcon from '@mui/icons-material/Tag';
import TelegramIcon from '@mui/icons-material/Send';

const webhookTemplate = 'https://YOUR_DOMAIN/hook/{provider_type}/{project_id_prefix}';

const Step = ({ num, title, children }: { num: number; title: string; children: React.ReactNode }) => (
  <Box sx={{ mb: 3 }}>
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1 }}>
      <Box sx={{
        width: 28, height: 28, borderRadius: '50%',
        bgcolor: 'primary.main', color: 'primary.contrastText',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        fontSize: '0.85rem', fontWeight: 700,
      }}>
        {num}
      </Box>
      <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>{title}</Typography>
    </Box>
    <Box sx={{ pl: 5.5 }}>{children}</Box>
  </Box>
);

const CodeBlock = ({ children }: { children: string }) => (
  <Box sx={{
    p: 2, borderRadius: 1, my: 1,
    bgcolor: 'rgba(0,0,0,0.3)',
    border: '1px solid',
    borderColor: 'divider',
    fontFamily: '"JetBrains Mono", monospace',
    fontSize: '0.8rem',
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-all',
    overflowX: 'auto',
  }}>
    {children}
  </Box>
);

const GitLabGuide = () => (
  <Box>
    <Alert severity="info" sx={{ mb: 3 }}>
      Webhook URL 格式：<code>{webhookTemplate.replace('{provider_type}', 'gitlab')}</code>
    </Alert>

    <Step num={1} title="進入 GitLab 專案設定">
      <Typography variant="body2">
        前往你的 GitLab 專案 → <strong>Settings</strong> → <strong>Webhooks</strong>
      </Typography>
    </Step>

    <Step num={2} title="新增 Webhook">
      <Typography variant="body2" sx={{ mb: 1 }}>在 URL 欄位填入：</Typography>
      <CodeBlock>{`https://YOUR_DOMAIN/hook/gitlab/{project_id_prefix}`}</CodeBlock>
      <Typography variant="body2" sx={{ mt: 1 }}>
        將 <code>YOUR_DOMAIN</code> 替換為你的伺服器域名，<code>{'{project_id_prefix}'}</code> 替換為你在本系統中建立的 Project ID 前綴。
      </Typography>
    </Step>

    <Step num={3} title="設定 Secret Token">
      <Typography variant="body2">
        在 <strong>Secret token</strong> 欄位填入你在建立 Provider 時設定的 <code>webhook_secret</code>。
        這用於驗證請求來源的合法性。
      </Typography>
    </Step>

    <Step num={4} title="選擇觸發事件">
      <Typography variant="body2" sx={{ mb: 1 }}>
        勾選以下事件：
      </Typography>
      <Box component="ul" sx={{ pl: 2 }}>
        <li><Typography variant="body2"><strong>Note events</strong>（留言事件）— 這是主要的觸發方式</Typography></li>
        <li><Typography variant="body2"><strong>Merge request events</strong>（可選）— 用於 MR 相關的操作</Typography></li>
        <li><Typography variant="body2"><strong>Issue events</strong>（可選）— 用於 Issue 相關的操作</Typography></li>
      </Box>
    </Step>

    <Step num={5} title="測試連線">
      <Typography variant="body2">
        儲存後，點擊 <strong>Test</strong> 按鈕發送測試請求。
        在本系統的 Tasks 頁面確認是否收到測試事件。
      </Typography>
    </Step>
  </Box>
);

const SlackGuide = () => (
  <Box>
    <Alert severity="info" sx={{ mb: 3 }}>
      Webhook URL 格式：<code>{webhookTemplate.replace('{provider_type}', 'slack')}</code>
    </Alert>

    <Step num={1} title="建立 Slack App">
      <Typography variant="body2">
        前往 <strong>api.slack.com/apps</strong> → 點擊 <strong>Create New App</strong> → 選擇 <strong>From scratch</strong>
      </Typography>
    </Step>

    <Step num={2} title="設定 Event Subscriptions">
      <Typography variant="body2" sx={{ mb: 1 }}>
        在左側選單點擊 <strong>Event Subscriptions</strong> → 開啟 <strong>Enable Events</strong>
      </Typography>
      <Typography variant="body2" sx={{ mb: 1 }}>Request URL 填入：</Typography>
      <CodeBlock>{`https://YOUR_DOMAIN/hook/slack/{project_id_prefix}`}</CodeBlock>
      <Typography variant="body2" sx={{ mt: 1 }}>
        Slack 會自動發送驗證請求，確認 URL 可用。
      </Typography>
    </Step>

    <Step num={3} title="訂閱 Bot Events">
      <Typography variant="body2" sx={{ mb: 1 }}>
        在 Event Subscriptions 頁面下方，點擊 <strong>Subscribe to bot events</strong>，新增：
      </Typography>
      <Box component="ul" sx={{ pl: 2 }}>
        <li><Typography variant="body2"><code>app_mention</code> — 當有人 @你的 Bot 時觸發</Typography></li>
        <li><Typography variant="body2"><code>message.channels</code>（可選）— 監聽頻道訊息</Typography></li>
      </Box>
    </Step>

    <Step num={4} title="設定 Bot Token Scopes">
      <Typography variant="body2" sx={{ mb: 1 }}>
        前往 <strong>OAuth & Permissions</strong>，在 <strong>Bot Token Scopes</strong> 新增：
      </Typography>
      <Box component="ul" sx={{ pl: 2 }}>
        <li><Typography variant="body2"><code>app_mentions:read</code></Typography></li>
        <li><Typography variant="body2"><code>chat:write</code></Typography></li>
        <li><Typography variant="body2"><code>channels:history</code>（如果需要讀取歷史訊息）</Typography></li>
      </Box>
    </Step>

    <Step num={5} title="安裝到 Workspace">
      <Typography variant="body2">
        回到 <strong>OAuth & Permissions</strong>，點擊 <strong>Install to Workspace</strong>。
        複製產生的 <strong>Bot User OAuth Token</strong>（以 <code>xoxb-</code> 開頭），
        貼到 Provider 的 config 中。
      </Typography>
    </Step>

    <Step num={6} title="邀請 Bot 到頻道">
      <Typography variant="body2">
        在 Slack 頻道中輸入 <code>/invite @你的Bot名稱</code> 將 Bot 加入頻道。
      </Typography>
    </Step>
  </Box>
);

const TelegramGuide = () => (
  <Box>
    <Alert severity="info" sx={{ mb: 3 }}>
      Webhook URL 格式：<code>{webhookTemplate.replace('{provider_type}', 'telegram')}</code>
    </Alert>

    <Step num={1} title="透過 @BotFather 建立 Bot">
      <Typography variant="body2" sx={{ mb: 1 }}>
        在 Telegram 搜尋 <strong>@BotFather</strong>，發送以下指令：
      </Typography>
      <CodeBlock>{`/newbot`}</CodeBlock>
      <Typography variant="body2" sx={{ mt: 1 }}>
        依照提示設定 Bot 名稱和 username。完成後會收到一個 <strong>Bot Token</strong>。
      </Typography>
    </Step>

    <Step num={2} title="設定 Webhook">
      <Typography variant="body2" sx={{ mb: 1 }}>
        使用以下 API 呼叫來設定 Webhook：
      </Typography>
      <CodeBlock>{`curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://YOUR_DOMAIN/hook/telegram/{project_id_prefix}"}'`}</CodeBlock>
    </Step>

    <Step num={3} title="驗證 Webhook 設定">
      <Typography variant="body2" sx={{ mb: 1 }}>確認 Webhook 已正確設定：</Typography>
      <CodeBlock>{`curl "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo"`}</CodeBlock>
    </Step>

    <Step num={4} title="設定 Bot Token">
      <Typography variant="body2">
        將 BotFather 給你的 Token 填入本系統 Provider 的 config JSON 中，
        格式如：<code>{`{"bot_token": "123456:ABC-DEF..."}`}</code>
      </Typography>
    </Step>

    <Step num={5} title="測試">
      <Typography variant="body2">
        在 Telegram 中找到你的 Bot，發送任何訊息。
        在本系統的 Tasks 頁面確認是否收到事件。
      </Typography>
    </Step>
  </Box>
);

const tabPanels = [GitLabGuide, SlackGuide, TelegramGuide];

const Guides = () => {
  const [tab, setTab] = useState(0);
  const Panel = tabPanels[tab];

  return (
    <Box sx={{ p: { xs: 2, md: 3 } }}>
      <Title title="Integration Guides" />

      <Typography variant="h4" sx={{ mb: 0.5 }}>整合指南</Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
        依照以下步驟將各平台與 OpenCode Bot 連接
      </Typography>

      <Card>
        <Tabs
          value={tab}
          onChange={(_, v) => setTab(v)}
          sx={{ borderBottom: 1, borderColor: 'divider', px: 2 }}
        >
          <Tab icon={<GitLabIcon />} iconPosition="start" label="GitLab" />
          <Tab icon={<SlackIcon />} iconPosition="start" label="Slack" />
          <Tab icon={<TelegramIcon />} iconPosition="start" label="Telegram" />
        </Tabs>
        <CardContent sx={{ p: 3 }}>
          <Panel />
        </CardContent>
      </Card>
    </Box>
  );
};

export default Guides;
