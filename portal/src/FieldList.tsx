import React, { useCallback } from "react";
import { ActionButton, IconButton, Stack, Text } from "@fluentui/react";
import { FormattedMessage } from "@oursky/react-messageformat";

import { useSystemConfig } from "./context/SystemConfigContext";
import { useFormField } from "./error/FormFieldContext";

import styles from "./FieldList.module.scss";

type RenderFieldListItem<T> = (
  index: number,
  value: T,
  onChange: (value: T) => void
) => React.ReactElement;

interface FieldListProps<T> {
  className?: string;
  label?: React.ReactNode;
  jsonPointer: RegExp | string;
  parentJSONPointer: RegExp | string;
  fieldName: string;
  fieldNameMessageID?: string;
  list: T[];
  onListChange: (list: T[]) => void;
  makeDefaultItem: () => T;
  renderListItem: RenderFieldListItem<T>;
  addButtonLabelMessageID?: string;
  errorMessage?: string;
}

const FieldList = function FieldList<T>(
  props: FieldListProps<T>
): React.ReactElement {
  const {
    className,
    label,
    jsonPointer,
    parentJSONPointer,
    fieldName,
    fieldNameMessageID,
    list,
    onListChange,
    renderListItem,
    makeDefaultItem,
    addButtonLabelMessageID,
    errorMessage: errorMessageProps,
  } = props;

  const { themes } = useSystemConfig();

  const { errorMessage } = useFormField(
    jsonPointer,
    parentJSONPointer,
    fieldName,
    fieldNameMessageID
  );

  const onItemChange = useCallback(
    (index: number, newValue: T) => {
      const newList = list.slice();
      newList[index] = newValue;
      onListChange(newList);
    },
    [onListChange, list]
  );

  const onItemAdd = useCallback(() => {
    const newList = list.slice();
    newList.push(makeDefaultItem());
    onListChange(newList);
  }, [list, onListChange, makeDefaultItem]);

  const onItemDelete = useCallback(
    (index: number) => {
      const newList = list.slice();
      newList.splice(index, 1);
      onListChange(newList);
    },
    [onListChange, list]
  );

  return (
    <section className={className}>
      {label ?? null}
      <Stack className={styles.list} tokens={{ childrenGap: 10 }}>
        {list.map((value, index) => (
          <FieldListItem
            key={index}
            index={index}
            value={value}
            onItemChange={onItemChange}
            onItemDelete={onItemDelete}
            renderListItem={renderListItem}
          />
        ))}
      </Stack>
      <Text className={styles.errorMessage}>
        {errorMessageProps ?? errorMessage}
      </Text>
      <ActionButton
        className={styles.addButton}
        theme={themes.actionButton}
        iconProps={{ iconName: "CirclePlus", className: styles.addButtonIcon }}
        onClick={onItemAdd}
      >
        <FormattedMessage id={addButtonLabelMessageID ?? "add"} />
      </ActionButton>
    </section>
  );
};

interface FieldListItemProps<T> {
  index: number;
  value: T;
  onItemChange: (index: number, newValue: T) => void;
  onItemDelete: (index: number) => void;
  renderListItem: RenderFieldListItem<T>;
}

function FieldListItem<T>(props: FieldListItemProps<T>) {
  const { index, value, onItemChange, onItemDelete, renderListItem } = props;
  const { themes } = useSystemConfig();

  const onChange = useCallback((newValue: T) => onItemChange(index, newValue), [
    onItemChange,
    index,
  ]);
  const onDelete = useCallback(() => onItemDelete(index), [
    onItemDelete,
    index,
  ]);

  return (
    <div className={styles.listItem}>
      {renderListItem(index, value, onChange)}
      <IconButton
        className={styles.deleteButton}
        onClick={onDelete}
        iconProps={{ iconName: "Delete" }}
        theme={themes.destructive}
      />
    </div>
  );
}

export default FieldList;
