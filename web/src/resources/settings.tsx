import { useState } from 'react';
import {
  List,
  DeleteButton,
  EditButton,
  Edit,
  SimpleForm,
  TextInput,
  TextField,
  Show,
  SimpleShowLayout,
  usePermissions,
  FunctionField,
  useRecordContext,
  Create,
  useListContext,
  BooleanInput,
  required,
  Button as RaButton,
  Link,
} from 'react-admin';
import { useFormContext } from 'react-hook-form';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableRow,
  Tooltip,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import Editor from '@monaco-editor/react';

type SettingType = 'text' | 'duration' | 'number' | 'multiline' | 'json' | 'boolean';

interface SettingConfig {
  key: string;
  label: string;
  type: SettingType;
  description?: string;
}

const SETTING_CATEGORIES: Record<string, SettingConfig[]> = {
  'Analyzer': [
    { key: 'analyzer_timeout', label: 'CLI Timeout', type: 'duration', description: 'Max execution time for OpenCode CLI (e.g., 5m, 10m)' },
    { key: 'analyzer_ack_template', label: 'Acknowledgment Template', type: 'multiline', description: 'Uses %s placeholders: mode, keyword, author' },
    { key: 'analyzer_error_template', label: 'Error Template', type: 'multiline', description: 'Uses %s for error message' },
    { key: 'analyzer_result_template', label: 'Result Template', type: 'multiline', description: 'Uses %s: result, mode, author' },
    { key: 'prompt_ask', label: 'Ask Mode Prompt', type: 'multiline', description: 'System prompt for ask mode' },
    { key: 'prompt_plan', label: 'Plan Mode Prompt', type: 'multiline', description: 'System prompt for plan mode' },
    { key: 'prompt_do', label: 'Do Mode Prompt', type: 'multiline', description: 'System prompt for do mode' },
    { key: 'prompt_default', label: 'Default Prompt', type: 'multiline', description: 'Fallback system prompt' },
    { key: 'prompt_format_suffix', label: 'Format Suffix', type: 'text', description: 'Appended to all prompts' },
  ],
  'Authentication': [
    { key: 'token_ttl', label: 'Token TTL', type: 'duration', description: 'Login session duration (e.g., 24h, 12h)' },
  ],
  'Providers': [
    { key: 'slack_http_timeout', label: 'Slack HTTP Timeout', type: 'duration', description: 'Timeout for Slack API calls' },
    { key: 'telegram_http_timeout', label: 'Telegram HTTP Timeout', type: 'duration', description: 'Timeout for Telegram API calls' },
    { key: 'telegram_parse_mode', label: 'Telegram Parse Mode', type: 'text', description: 'Markdown, MarkdownV2, or HTML' },
  ],
  'API': [
    { key: 'task_list_default_limit', label: 'Default Task Limit', type: 'number', description: 'Default page size for task list' },
    { key: 'task_list_max_limit', label: 'Max Task Limit', type: 'number', description: 'Maximum allowed page size' },
    { key: 'default_git_branch', label: 'Default Git Branch', type: 'text', description: 'Default branch for new projects' },
  ],
  'MCP': [
    { key: 'mcp_install_timeout', label: 'Install Timeout', type: 'duration', description: 'Timeout for npm package installation' },
    { key: 'mcp_uninstall_timeout', label: 'Uninstall Timeout', type: 'duration', description: 'Timeout for npm package removal' },
    { key: 'mcp_enabled', label: 'MCP Enabled', type: 'boolean', description: 'Enable/disable MCP protocol server' },
    { key: 'mcp_endpoint', label: 'MCP Endpoint', type: 'text', description: 'MCP server HTTP path' },
  ],
  'OpenCode Config': [
    { key: 'opencode_binary', label: 'Binary Path', type: 'text', description: 'Path to opencode CLI binary' },
    { key: 'opencode_auth_json', label: 'auth.json', type: 'json', description: 'API keys configuration' },
    { key: 'opencode_config_json', label: '.opencode.json', type: 'json', description: 'OpenCode configuration' },
    { key: 'opencode_ohmy_json', label: 'oh-my-opencode.json', type: 'json', description: 'Custom OpenCode settings' },
  ],
};

const SETTING_CONFIG_MAP = Object.values(SETTING_CATEGORIES).flat().reduce((acc, config) => {
  acc[config.key] = config;
  return acc;
}, {} as Record<string, SettingConfig>);

const JSON_CONFIG_KEYS = ['opencode_auth_json', 'opencode_config_json', 'opencode_ohmy_json'];
const isJsonConfig = (key: string) => JSON_CONFIG_KEYS.includes(key) || SETTING_CONFIG_MAP[key]?.type === 'json';

const parseValue = (value: string | undefined) => {
  if (value === undefined || value === null) return undefined;
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
};

const formatValueForDisplay = (value: string | undefined, type?: SettingType) => {
  if (value === undefined || value === null) return null;
  
  const parsed = parseValue(value);

  if (type === 'json' || isJsonConfig(String(value))) {
    return <Chip label="JSON Config" size="small" color="secondary" variant="outlined" />;
  }

  if (type === 'boolean' || typeof parsed === 'boolean') {
    return parsed ? 
      <Chip label="True" size="small" color="success" variant="filled" /> : 
      <Chip label="False" size="small" color="error" variant="filled" />;
  }

  if (type === 'multiline') {
    const str = String(parsed);
    const firstLine = str.split('\n')[0];
    return (
      <Tooltip title={<pre style={{ margin: 0 }}>{str}</pre>}>
        <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
          {firstLine.length > 50 ? firstLine.slice(0, 50) + '...' : firstLine}
          {str.split('\n').length > 1 ? ' [...]' : ''}
        </Typography>
      </Tooltip>
    );
  }

  return (
    <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
      {String(parsed)}
    </Typography>
  );
};

const MonacoJsonEditor = ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
  <Box sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
    <Editor
      height="60vh"
      defaultLanguage="json"
      value={value}
      onChange={(v) => onChange(v || '')}
      theme="vs-dark"
      options={{
        minimap: { enabled: true },
        fontSize: 13,
        fontFamily: '"JetBrains Mono", monospace',
        lineNumbers: 'on',
        scrollBeyondLastLine: false,
        tabSize: 2,
        wordWrap: 'on',
        formatOnPaste: true,
      }}
    />
  </Box>
);

const CategorizedLayout = () => {
  const { data, isLoading } = useListContext();
  const { permissions } = usePermissions();

  if (isLoading) return null;

  const dataMap = (data || []).reduce((acc, record) => {
    acc[record.key] = record;
    return acc;
  }, {} as Record<string, { key: string; value: string; id?: string }>);

  const knownKeys = new Set(Object.keys(SETTING_CONFIG_MAP));
  const otherKeys = (data || []).filter(r => !knownKeys.has(r.key)).map(r => r.key);

  const renderRow = (key: string, config?: SettingConfig) => {
    const record = dataMap[key];
    const exists = !!record;
    const label = config?.label || key;
    const description = config?.description;
    const type = config?.type;

    return (
      <TableRow key={key} hover>
        <TableCell sx={{ width: '30%', verticalAlign: 'top' }}>
          <Typography variant="subtitle2">{label}</Typography>
          {description && (
            <Typography variant="caption" color="text.secondary">
              {description}
            </Typography>
          )}
          {!config && <Typography variant="caption" color="text.secondary">{key}</Typography>}
        </TableCell>
        <TableCell sx={{ width: '50%', verticalAlign: 'top' }}>
          {exists ? (
            type === 'json' || isJsonConfig(key) ? 
              <Chip label="JSON Config" size="small" color="secondary" variant="outlined" /> :
              formatValueForDisplay(record.value, type)
          ) : (
            <Typography variant="body2" color="text.disabled" sx={{ fontStyle: 'italic' }}>
              Not set (using default)
            </Typography>
          )}
        </TableCell>
        <TableCell align="right" sx={{ verticalAlign: 'top' }}>
          {permissions === 'admin' && (
            <Box display="flex" justifyContent="flex-end" gap={1}>
              {exists ? (
                <>
                  <EditButton record={record} label="" />
                  <DeleteButton record={record} label="" mutationMode="pessimistic" />
                </>
              ) : (
                <RaButton
                  label="Set"
                  component={Link}
                  to={{
                    pathname: '/settings/create',
                    search: `?source=${JSON.stringify({ key })}`,
                  }}
                  startIcon={<AddIcon />}
                  size="small"
                />
              )}
            </Box>
          )}
        </TableCell>
      </TableRow>
    );
  };

  return (
    <Box sx={{ mt: 2 }}>
      <Grid container spacing={3}>
        {Object.entries(SETTING_CATEGORIES).map(([category, configs]) => (
          <Grid size={12} key={category}>
            <Card elevation={2}>
              <CardContent sx={{ pb: 1 }}>
                <Typography variant="h6" gutterBottom color="primary">
                  {category}
                </Typography>
                <TableContainer>
                  <Table size="small">
                    <TableBody>
                      {configs.map(config => renderRow(config.key, config))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </CardContent>
            </Card>
          </Grid>
        ))}

        {otherKeys.length > 0 && (
          <Grid size={12}>
            <Card elevation={2}>
              <CardContent>
                <Typography variant="h6" gutterBottom color="text.secondary">
                  Other Settings
                </Typography>
                <TableContainer>
                  <Table size="small">
                    <TableBody>
                      {otherKeys.map(key => renderRow(key))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </CardContent>
            </Card>
          </Grid>
        )}
      </Grid>
    </Box>
  );
};

export const SettingsList = () => (
  <List 
    sort={{ field: 'key', order: 'ASC' }} 
    perPage={100} 
    pagination={false}
    component="div"
  >
    <CategorizedLayout />
  </List>
);

export const SettingsCreate = () => (
  <Create redirect="list">
    <SimpleForm>
      <TextInput source="key" fullWidth validate={required()} />
      <TextInput 
        source="value" 
        fullWidth 
        multiline 
        minRows={3} 
        helperText='Enter value as JSON (e.g. "some string", 50, true)' 
      />
    </SimpleForm>
  </Create>
);

interface SettingValueInputProps {
  jsonValue: string | null;
  setJsonValue: (v: string | null) => void;
}

const SettingValueInput = ({ jsonValue, setJsonValue }: SettingValueInputProps) => {
  const record = useRecordContext();
  const { setValue } = useFormContext();
  
  if (!record) return null;
  const key = String(record.key || record.id);
  const config = SETTING_CONFIG_MAP[key];
  const type = config?.type || 'text';
  
  if (type === 'json' || isJsonConfig(key)) {
    const initial = jsonValue ?? String(record.value ?? '{}');
    return (
      <Box sx={{ width: '100%', mt: 2 }}>
        <Typography variant="subtitle2" sx={{ mb: 1 }}>
          JSON Configuration
        </Typography>
        <MonacoJsonEditor
          value={initial}
          onChange={(v) => {
            setJsonValue(v);
            setValue('value', v, { shouldDirty: true });
          }}
        />
      </Box>
    );
  }

  if (type === 'boolean') {
    return (
      <BooleanInput 
        source="value" 
        format={(v: string) => {
          try { return JSON.parse(v); } catch { return false; }
        }}
        parse={(v: boolean) => JSON.stringify(v)}
      />
    );
  }

  const commonProps = {
    source: "value",
    fullWidth: true,
    helperText: config?.description,
    format: (v: string) => {
      try { return JSON.parse(v); } catch { return v; }
    },
    parse: (v: unknown) => JSON.stringify(v),
  };

  if (type === 'multiline') {
    return <TextInput {...commonProps} multiline minRows={4} maxRows={12} />;
  }

  if (type === 'number') {
    return (
      <TextInput 
        {...commonProps} 
        type="number" 
        parse={(v) => {
          if (v === '') return undefined;
          const n = Number(v);
          return isNaN(n) ? JSON.stringify(v) : JSON.stringify(n);
        }} 
      />
    );
  }

  if (type === 'duration') {
    return <TextInput {...commonProps} helperText={config?.description || "e.g., 5m, 1h"} />;
  }

  return <TextInput {...commonProps} />;
};

export const SettingsEdit = () => {
  const [jsonValue, setJsonValue] = useState<string | null>(null);

  return (
    <Edit
      mutationMode="pessimistic"
      transform={(data: Record<string, unknown>) => {
        if (jsonValue !== null) {
          return { ...data, value: jsonValue };
        }
        return data;
      }}
    >
      <SimpleForm>
        <TextInput source="key" fullWidth disabled />
        <SettingValueInput jsonValue={jsonValue} setJsonValue={setJsonValue} />
      </SimpleForm>
    </Edit>
  );
};

export const SettingsShow = () => (
  <Show>
    <SimpleShowLayout>
      <TextField source="key" />
      <FunctionField
        label="Value"
        render={(record: Record<string, unknown>) => {
          const key = String(record.key || record.id);
          const val = String(record.value ?? '');
          if (isJsonConfig(key)) {
            return (
              <Box sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, overflow: 'hidden' }}>
                <Editor
                  height="50vh"
                  defaultLanguage="json"
                  value={val}
                  theme="vs-dark"
                  options={{
                    readOnly: true,
                    minimap: { enabled: true },
                    fontSize: 13,
                    fontFamily: '"JetBrains Mono", monospace',
                  }}
                />
              </Box>
            );
          }
          return (
            <Typography sx={{ fontFamily: '"JetBrains Mono", monospace', whiteSpace: 'pre-wrap' }}>
              {val}
            </Typography>
          );
        }}
      />
    </SimpleShowLayout>
  </Show>
);
