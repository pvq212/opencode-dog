import {
  List, Datagrid, TextField, BooleanField, EditButton, ShowButton, DeleteButton,
  Create, Edit, SimpleForm, TextInput, BooleanInput, ReferenceInput, SelectInput,
  Show, SimpleShowLayout, TabbedShowLayout,
  usePermissions, TopToolbar, CreateButton, ExportButton, FilterButton,
  ReferenceManyField, FunctionField,
} from 'react-admin';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';

const ProjectFilters = [
  <TextInput key="q" source="q" label="Search" alwaysOn />,
];

const ProjectListActions = () => {
  const { permissions } = usePermissions();
  return (
    <TopToolbar>
      <FilterButton />
      {permissions !== 'viewer' && <CreateButton />}
      <ExportButton />
    </TopToolbar>
  );
};

export const ProjectList = () => {
  const { permissions } = usePermissions();
  return (
    <List filters={ProjectFilters} actions={<ProjectListActions />} sort={{ field: 'name', order: 'ASC' }}>
      <Datagrid bulkActionButtons={false}>
        <TextField source="name" />
        <TextField source="ssh_url" label="SSH URL" />
        <TextField source="default_branch" label="Branch" />
        <FunctionField
          label="Status"
          render={(record: Record<string, unknown>) => (
            <Chip
              label={record.enabled ? 'Active' : 'Disabled'}
              color={record.enabled ? 'success' : 'default'}
              size="small"
              variant="outlined"
            />
          )}
        />
        <ShowButton />
        {permissions !== 'viewer' && <EditButton />}
        {permissions === 'admin' && <DeleteButton />}
      </Datagrid>
    </List>
  );
};

export const ProjectCreate = () => (
  <Create redirect="show">
    <SimpleForm>
      <TextInput source="name" fullWidth isRequired />
      <TextInput source="ssh_url" label="SSH URL" fullWidth isRequired />
      <ReferenceInput source="ssh_key_id" reference="ssh-keys">
        <SelectInput optionText="name" label="SSH Key" fullWidth />
      </ReferenceInput>
      <TextInput source="default_branch" label="Default Branch" defaultValue="main" fullWidth />
      <BooleanInput source="enabled" defaultValue={true} />
    </SimpleForm>
  </Create>
);

export const ProjectEdit = () => (
  <Edit>
    <SimpleForm>
      <TextInput source="name" fullWidth isRequired />
      <TextInput source="ssh_url" label="SSH URL" fullWidth isRequired />
      <ReferenceInput source="ssh_key_id" reference="ssh-keys">
        <SelectInput optionText="name" label="SSH Key" fullWidth />
      </ReferenceInput>
      <TextInput source="default_branch" label="Default Branch" fullWidth />
      <BooleanInput source="enabled" />
    </SimpleForm>
  </Edit>
);

export const ProjectShow = () => (
  <Show>
    <TabbedShowLayout>
      <TabbedShowLayout.Tab label="Details">
        <SimpleShowLayout>
          <TextField source="id" />
          <TextField source="name" />
          <TextField source="ssh_url" label="SSH URL" />
          <TextField source="default_branch" label="Default Branch" />
          <BooleanField source="enabled" />
          <TextField source="created_at" label="Created" />
        </SimpleShowLayout>
      </TabbedShowLayout.Tab>

      <TabbedShowLayout.Tab label="Providers">
        <Box sx={{ mt: 1 }}>
          <ReferenceManyField reference="providers" target="project_id" label={false}>
            <Datagrid bulkActionButtons={false}>
              <FunctionField
                label="Type"
                render={(record: Record<string, unknown>) => (
                  <Chip
                    label={String(record.provider_type || '').toUpperCase()}
                    size="small"
                    color={
                      record.provider_type === 'gitlab' ? 'warning' :
                      record.provider_type === 'slack' ? 'info' :
                      record.provider_type === 'telegram' ? 'primary' : 'default'
                    }
                    variant="filled"
                  />
                )}
              />
              <TextField source="webhook_path" label="Webhook Path" />
              <BooleanField source="enabled" />
            </Datagrid>
          </ReferenceManyField>
        </Box>
      </TabbedShowLayout.Tab>

      <TabbedShowLayout.Tab label="Keywords">
        <Box sx={{ mt: 1 }}>
          <ReferenceManyField reference="keywords" target="project_id" label={false}>
            <Datagrid bulkActionButtons={false}>
              <TextField source="keyword" />
              <FunctionField
                label="Mode"
                render={(record: Record<string, unknown>) => (
                  <Chip
                    label={String(record.mode)}
                    size="small"
                    color={
                      record.mode === 'do' ? 'error' :
                      record.mode === 'plan' ? 'warning' : 'info'
                    }
                    variant="outlined"
                  />
                )}
              />
            </Datagrid>
          </ReferenceManyField>
        </Box>
      </TabbedShowLayout.Tab>
    </TabbedShowLayout>
  </Show>
);
