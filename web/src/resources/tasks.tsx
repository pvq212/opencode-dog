import {
  List, Datagrid, TextField, DateField, ShowButton,
  Show, SimpleShowLayout,
  FunctionField, SelectInput,
} from 'react-admin';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Typography from '@mui/material/Typography';
import Link from '@mui/material/Link';
import ReactMarkdown from 'react-markdown';

const statusColors: Record<string, 'success' | 'warning' | 'info' | 'error' | 'default'> = {
  completed: 'success',
  processing: 'warning',
  pending: 'info',
  failed: 'error',
};

const providerColors: Record<string, 'warning' | 'info' | 'primary' | 'default'> = {
  gitlab: 'warning',
  slack: 'info',
  telegram: 'primary',
};

const TaskFilters = [
  <SelectInput key="status" source="status" alwaysOn choices={[
    { id: 'pending', name: 'Pending' },
    { id: 'processing', name: 'Processing' },
    { id: 'completed', name: 'Completed' },
    { id: 'failed', name: 'Failed' },
  ]} />,
  <SelectInput key="provider_type" source="provider_type" choices={[
    { id: 'gitlab', name: 'GitLab' },
    { id: 'slack', name: 'Slack' },
    { id: 'telegram', name: 'Telegram' },
  ]} />,
];

export const TaskList = () => (
  <List filters={TaskFilters} sort={{ field: 'created_at', order: 'DESC' }}>
    <Datagrid bulkActionButtons={false}>
      <FunctionField
        label="Status"
        render={(record: Record<string, unknown>) => (
          <Chip
            label={String(record.status)}
            color={statusColors[String(record.status)] || 'default'}
            size="small"
            variant="outlined"
          />
        )}
      />
      <FunctionField
        label="Provider"
        render={(record: Record<string, unknown>) => (
          <Chip
            label={String(record.provider_type || '').toUpperCase()}
            size="small"
            color={providerColors[String(record.provider_type)] || 'default'}
            variant="filled"
          />
        )}
      />
      <TextField source="title" />
      <TextField source="author" />
      <FunctionField
        label="Mode"
        render={(record: Record<string, unknown>) => (
          <Chip
            label={String(record.trigger_mode || '—')}
            size="small"
            variant="outlined"
            color={record.trigger_mode === 'do' ? 'error' : record.trigger_mode === 'plan' ? 'warning' : 'info'}
          />
        )}
      />
      <TextField source="trigger_keyword" label="Keyword" />
      <FunctionField
        label="External"
        render={(record: Record<string, unknown>) =>
          record.external_ref ? (
            <Link href={String(record.external_ref)} target="_blank" rel="noopener" sx={{ fontFamily: '"JetBrains Mono", monospace', fontSize: '0.75rem' }}>
              Link ↗
            </Link>
          ) : '—'
        }
      />
      <DateField source="created_at" label="Created" showTime />
      <ShowButton />
    </Datagrid>
  </List>
);

export const TaskShow = () => (
  <Show>
    <SimpleShowLayout>
      <TextField source="id" />
      <FunctionField
        label="Status"
        render={(record: Record<string, unknown>) => (
          <Chip
            label={String(record.status)}
            color={statusColors[String(record.status)] || 'default'}
            size="small"
          />
        )}
      />
      <FunctionField
        label="Provider"
        render={(record: Record<string, unknown>) => (
          <Chip
            label={String(record.provider_type || '').toUpperCase()}
            size="small"
            color={providerColors[String(record.provider_type)] || 'default'}
          />
        )}
      />
      <TextField source="title" />
      <TextField source="author" />
      <TextField source="trigger_mode" label="Mode" />
      <TextField source="trigger_keyword" label="Keyword" />
      <FunctionField
        label="External Reference"
        render={(record: Record<string, unknown>) =>
          record.external_ref ? (
            <Link href={String(record.external_ref)} target="_blank" rel="noopener">
              {String(record.external_ref)}
            </Link>
          ) : '—'
        }
      />
      <DateField source="created_at" label="Created" showTime />
      <DateField source="updated_at" label="Updated" showTime />

      <FunctionField
        label="Result"
        render={(record: Record<string, unknown>) =>
          record.result ? (
            <Box sx={{
              p: 2, mt: 1, borderRadius: 1,
              bgcolor: 'background.default',
              border: '1px solid',
              borderColor: 'divider',
              '& pre': { fontFamily: '"JetBrains Mono", monospace', fontSize: '0.8rem', overflow: 'auto' },
              '& code': { fontFamily: '"JetBrains Mono", monospace', fontSize: '0.8rem' },
            }}>
              <ReactMarkdown>{String(record.result)}</ReactMarkdown>
            </Box>
          ) : (
            <Typography color="text.secondary">No result yet</Typography>
          )
        }
      />

      <FunctionField
        label="Error"
        render={(record: Record<string, unknown>) =>
          record.error_message ? (
            <Box sx={{ p: 2, mt: 1, borderRadius: 1, bgcolor: 'error.main', color: 'error.contrastText' }}>
              <Typography variant="body2" sx={{ fontFamily: '"JetBrains Mono", monospace' }}>
                {String(record.error_message)}
              </Typography>
            </Box>
          ) : null
        }
      />
    </SimpleShowLayout>
  </Show>
);
