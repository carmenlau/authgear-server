import React, { useCallback, useContext, useMemo, useState } from "react";
import {
  IColumn,
  Checkbox,
  SelectionMode,
  ICheckboxProps,
  DefaultEffects,
  Text,
  Dropdown,
  Toggle,
} from "@fluentui/react";
import produce from "immer";
import deepEqual from "deep-equal";
import { Context, FormattedMessage } from "@oursky/react-messageformat";

import DetailsListWithOrdering from "../../DetailsListWithOrdering";
import { swap } from "../../OrderButtons";
import NavigationBlockerDialog from "../../NavigationBlockerDialog";
import ButtonWithLoading from "../../ButtonWithLoading";
import ErrorDialog from "../../error/ErrorDialog";
import { ModifiedIndicatorPortal } from "../../ModifiedIndicatorPortal";
import FormTextField from "../../FormTextField";
import ShowUnhandledValidationErrorCause from "../../error/ShowUnhandledValidationErrorCauses";
import {
  PortalAPIAppConfig,
  primaryAuthenticatorTypes,
  secondaryAuthenticatorTypes,
  PrimaryAuthenticatorType,
  SecondaryAuthenticatorType,
  secondaryAuthenticationModes,
  SecondaryAuthenticationMode,
  PortalAPIApp,
} from "../../types";
import {
  useDropdown,
  useIntegerTextField,
  useCheckbox,
} from "../../hook/useInput";
import {
  isArrayEqualInOrder,
  clearEmptyObject,
  setFieldIfChanged,
  setNumericFieldIfChanged,
} from "../../util/misc";
import { FormContext } from "../../error/FormContext";
import { useValidationError } from "../../error/useValidationError";
import {
  GenericErrorHandlingRule,
  useGenericError,
} from "../../error/useGenericError";

import styles from "./AuthenticationAuthenticatorSettings.module.scss";

interface Props {
  effectiveAppConfig: PortalAPIAppConfig | null;
  rawAppConfig: PortalAPIAppConfig | null;
  updateAppConfig: (
    appConfig: PortalAPIAppConfig
  ) => Promise<PortalAPIApp | null>;
  updatingAppConfig: boolean;
  updateAppConfigError: unknown;
  resetForm: () => void;
}

interface AuthenticatorCheckboxProps extends ICheckboxProps {
  authenticatorKey: string;
  onAuthticatorCheckboxChange: (key: string, checked: boolean) => void;
}

interface AuthenticatorListItem<KeyType> {
  activated: boolean;
  key: KeyType;
}

interface AuthenticatorsState {
  primaryAuthenticators: AuthenticatorListItem<PrimaryAuthenticatorType>[];
  secondaryAuthenticators: AuthenticatorListItem<SecondaryAuthenticatorType>[];
}

interface PolicySectionState {
  secondaryAuthenticationMode: SecondaryAuthenticationMode;
  recoveryCodeNumber: string;
  allowRetrieveRecoveryCode: boolean;
}

interface AuthenticationAuthenticatorScreenState
  extends AuthenticatorsState,
    PolicySectionState {}

const ALL_REQUIRE_MFA_OPTIONS: SecondaryAuthenticationMode[] = [
  ...secondaryAuthenticationModes,
];
const HIDDEN_REQUIRE_MFA_OPTIONS: SecondaryAuthenticationMode[] = [
  "if_requested",
];

const primaryAuthenticatorTypeLocaleKey: Record<string, string> = {
  oob_otp: "AuthenticatorType.primary.oob-otp",
  password: "AuthenticatorType.primary.password",
};

const secondaryAuthenticatorTypeLocalKey: Record<string, string> = {
  totp: "AuthenticatorType.secondary.totp",
  oob_otp: "AuthenticatorType.secondary.oob-otp",
  password: "AuthenticatorType.secondary.password",
};

const authenticatorTypeLocaleKey = {
  primary: primaryAuthenticatorTypeLocaleKey,
  secondary: secondaryAuthenticatorTypeLocalKey,
};

const AuthenticatorCheckbox: React.FC<AuthenticatorCheckboxProps> = function AuthenticatorCheckbox(
  props: AuthenticatorCheckboxProps
) {
  const onChange = React.useCallback(
    (_event, checked?: boolean) => {
      props.onAuthticatorCheckboxChange(props.authenticatorKey, !!checked);
    },
    [props]
  );

  return <Checkbox {...props} onChange={onChange} />;
};

function useRenderItemColumn(
  kind: "primary" | "secondary",
  onCheckboxClicked: (key: string, checked: boolean) => void
) {
  const { renderToString } = useContext(Context);

  const renderItemColumn = React.useCallback(
    (
      item: AuthenticatorListItem<
        PrimaryAuthenticatorType | SecondaryAuthenticatorType
      >,
      _index?: number,
      column?: IColumn
    ) => {
      switch (column?.key) {
        case "activated":
          return (
            <AuthenticatorCheckbox
              ariaLabel={item.key}
              authenticatorKey={item.key}
              checked={item.activated}
              onAuthticatorCheckboxChange={onCheckboxClicked}
            />
          );

        case "key": {
          const authenticatorName = renderToString(
            authenticatorTypeLocaleKey[kind][item.key]
          );
          return <span>{authenticatorName}</span>;
        }

        default:
          return <span>{item.key}</span>;
      }
    },
    [onCheckboxClicked, kind, renderToString]
  );
  return renderItemColumn;
}

function useOnActivateClicked<KeyType extends string>(
  state: AuthenticatorListItem<KeyType>[],
  setState: (
    stateUpdater: (
      prev: AuthenticatorListItem<KeyType>[]
    ) => AuthenticatorListItem<KeyType>[]
  ) => void
) {
  const onActivateClicked = React.useCallback(
    (key: string, checked: boolean) => {
      const itemIndex = state.findIndex(
        (authenticator) => authenticator.key === key
      );
      if (itemIndex < 0) {
        return;
      }
      setState((prev: AuthenticatorListItem<KeyType>[]) => {
        const newState = produce(prev, (draftState) => {
          draftState[itemIndex].activated = checked;
        });
        return newState;
      });
    },
    [state, setState]
  );
  return onActivateClicked;
}

// return list with all keys, active key from config in order
function makeAuthenticatorKeys<KeyType>(
  activeKeys: KeyType[],
  availableKeys: KeyType[]
) {
  const activeKeySet = new Set(activeKeys);
  const inactiveKeys = availableKeys.filter((key) => !activeKeySet.has(key));
  return [...activeKeys, ...inactiveKeys].map((key) => {
    return {
      activated: activeKeySet.has(key),
      key,
    };
  });
}

const constructAuthenticatorListData = (
  appConfig: PortalAPIAppConfig | null
): AuthenticatorsState => {
  const authentication = appConfig?.authentication;

  const primaryAuthenticators = makeAuthenticatorKeys(
    authentication?.primary_authenticators ?? [],
    [...primaryAuthenticatorTypes]
  );
  const secondaryAuthenticators = makeAuthenticatorKeys(
    authentication?.secondary_authenticators ?? [],
    [...secondaryAuthenticatorTypes]
  );

  return {
    primaryAuthenticators,
    secondaryAuthenticators,
  };
};

function getActivatedKeyListFromState<KeyType>(
  state: AuthenticatorListItem<KeyType>[]
) {
  return state
    .filter((authenticator) => authenticator.activated)
    .map((authenticator) => authenticator.key);
}

function getEffectiveState(state: AuthenticationAuthenticatorScreenState) {
  const {
    primaryAuthenticators,
    secondaryAuthenticators,
    ...reducedState
  } = state;

  return {
    activePrimaryAuthenticatorKeylist: getActivatedKeyListFromState(
      primaryAuthenticators
    ),
    activatedSecondaryAuthenticatorKeyList: getActivatedKeyListFromState(
      secondaryAuthenticators
    ),
    ...reducedState,
  };
}

function isStateEquivalent(
  state1: AuthenticationAuthenticatorScreenState,
  state2: AuthenticationAuthenticatorScreenState
): boolean {
  return deepEqual(getEffectiveState(state1), getEffectiveState(state2), {
    strict: true,
  });
}

const AuthenticationAuthenticatorSettings: React.FC<Props> = function AuthenticationAuthenticatorSettings(
  props: Props
) {
  const {
    effectiveAppConfig,
    rawAppConfig,
    updateAppConfig,
    updatingAppConfig,
    updateAppConfigError,
    resetForm,
  } = props;
  const { renderToString } = React.useContext(Context);

  const authenticatorColumns: IColumn[] = [
    {
      key: "activated",
      fieldName: "activated",
      name: renderToString("AuthenticationAuthenticator.activateHeader"),
      className: styles.authenticatorColumn,
      minWidth: 120,
      maxWidth: 120,
    },
    {
      key: "key",
      fieldName: "key",
      name: renderToString("AuthenticationAuthenticator.authenticatorHeader"),
      className: styles.authenticatorColumn,
      minWidth: 300,
      maxWidth: 300,
    },
  ];

  const initialAuthenticatorsState: AuthenticatorsState = useMemo(() => {
    return constructAuthenticatorListData(effectiveAppConfig);
  }, [effectiveAppConfig]);

  const initialPolicySectionState: PolicySectionState = useMemo(() => {
    const authenticationConfig = effectiveAppConfig?.authentication;
    return {
      secondaryAuthenticationMode:
        authenticationConfig?.secondary_authentication_mode ?? "if_exists",
      recoveryCodeNumber:
        authenticationConfig?.recovery_code?.count?.toString() ?? "",
      allowRetrieveRecoveryCode: !!authenticationConfig?.recovery_code
        ?.list_enabled,
    };
  }, [effectiveAppConfig]);

  const initialState: AuthenticationAuthenticatorScreenState = useMemo(() => {
    return {
      ...initialAuthenticatorsState,
      ...initialPolicySectionState,
    };
  }, [initialAuthenticatorsState, initialPolicySectionState]);

  const [state, setState] = useState<AuthenticationAuthenticatorScreenState>(
    initialState
  );

  const {
    primaryAuthenticators,
    secondaryAuthenticators,
    secondaryAuthenticationMode,
    recoveryCodeNumber,
    allowRetrieveRecoveryCode,
  } = state;

  const isFormModified = useMemo(() => {
    return !isStateEquivalent(initialState, state);
  }, [initialState, state]);

  const displaySecondaryAuthenticatorMode = useCallback(
    (key: SecondaryAuthenticationMode) => {
      const messageIdMap: Record<SecondaryAuthenticationMode, string> = {
        required: "AuthenticationAuthenticator.policy.require-mfa.required",
        if_exists: "AuthenticationAuthenticator.policy.require-mfa.if-exists",
        if_requested:
          "AuthenticationAuthenticator.policy.require-mfa.if-requested",
      };

      return renderToString(messageIdMap[key]);
    },
    [renderToString]
  );

  const {
    options: requireMFAOptions,
    onChange: onRequireMFAOptionChange,
  } = useDropdown(
    ALL_REQUIRE_MFA_OPTIONS,
    (option) => {
      setState((prev) => ({
        ...prev,
        secondaryAuthenticationMode: option,
      }));
    },
    secondaryAuthenticationMode,
    displaySecondaryAuthenticatorMode,
    // NOTE: not supported yet
    new Set(HIDDEN_REQUIRE_MFA_OPTIONS)
  );

  const { onChange: onRecoveryCodeNumberChange } = useIntegerTextField(
    (value) => {
      setState((prev) => ({
        ...prev,
        recoveryCodeNumber: value,
      }));
    }
  );

  const { onChange: onAllowRetrieveRecoveryCodeChange } = useCheckbox(
    (checked: boolean) => {
      setState((prev) => ({
        ...prev,
        allowRetrieveRecoveryCode: checked,
      }));
    }
  );

  const onPrimarySwapClicked = React.useCallback(
    (index1: number, index2: number) => {
      setState((prev) => ({
        ...prev,
        primaryAuthenticators: swap(prev.primaryAuthenticators, index1, index2),
      }));
    },
    []
  );
  const onSecondarySwapClicked = React.useCallback(
    (index1: number, index2: number) => {
      setState((prev) => ({
        ...prev,
        secondaryAuthenticators: swap(
          prev.secondaryAuthenticators,
          index1,
          index2
        ),
      }));
    },
    []
  );

  const onPrimaryActivateClicked = useOnActivateClicked(
    primaryAuthenticators,
    (stateUpdater) => {
      setState((prev) => ({
        ...prev,
        primaryAuthenticators: stateUpdater(prev.primaryAuthenticators),
      }));
    }
  );
  const onSecondaryActivateClicked = useOnActivateClicked(
    secondaryAuthenticators,
    (stateUpdater) => {
      setState((prev) => ({
        ...prev,
        secondaryAuthenticators: stateUpdater(prev.secondaryAuthenticators),
      }));
    }
  );

  const renderPrimaryItemColumn = useRenderItemColumn(
    "primary",
    onPrimaryActivateClicked
  );
  const renderSecondaryItemColumn = useRenderItemColumn(
    "secondary",
    onSecondaryActivateClicked
  );

  const renderPrimaryAriaLabel = React.useCallback(
    (index?: number): string => {
      return index != null ? primaryAuthenticators[index].key : "";
    },
    [primaryAuthenticators]
  );
  const renderSecondaryAriaLabel = React.useCallback(
    (index?: number): string => {
      return index != null ? secondaryAuthenticators[index].key : "";
    },
    [secondaryAuthenticators]
  );

  const onFormSubmit = React.useCallback(
    (ev: React.SyntheticEvent<HTMLElement>) => {
      ev.preventDefault();
      ev.stopPropagation();

      if (effectiveAppConfig == null || rawAppConfig == null) {
        return;
      }

      const initialActivatedPrimaryKeyList =
        effectiveAppConfig.authentication?.primary_authenticators ?? [];
      const initialActivatedSecondaryKeyList =
        effectiveAppConfig.authentication?.secondary_authenticators ?? [];

      const activatedPrimaryKeyList = getActivatedKeyListFromState(
        state.primaryAuthenticators
      );
      const activatedSecondaryKeyList = getActivatedKeyListFromState(
        state.secondaryAuthenticators
      );

      const newAppConfig = produce(rawAppConfig, (draftConfig) => {
        draftConfig.authentication = draftConfig.authentication ?? {};
        const { authentication } = draftConfig;
        if (
          !isArrayEqualInOrder(
            initialActivatedPrimaryKeyList,
            activatedPrimaryKeyList
          )
        ) {
          authentication.primary_authenticators = activatedPrimaryKeyList;
        }
        if (
          !isArrayEqualInOrder(
            initialActivatedSecondaryKeyList,
            activatedSecondaryKeyList
          )
        ) {
          authentication.secondary_authenticators = activatedSecondaryKeyList;
        }

        // Policy section
        authentication.recovery_code = authentication.recovery_code ?? {};

        setFieldIfChanged(
          authentication,
          "secondary_authentication_mode",
          initialState.secondaryAuthenticationMode,
          state.secondaryAuthenticationMode
        );

        setNumericFieldIfChanged(
          authentication.recovery_code,
          "count",
          initialState.recoveryCodeNumber,
          state.recoveryCodeNumber
        );

        setFieldIfChanged(
          authentication.recovery_code,
          "list_enabled",
          initialState.allowRetrieveRecoveryCode,
          state.allowRetrieveRecoveryCode
        );

        clearEmptyObject(draftConfig);
      });

      updateAppConfig(newAppConfig).catch(() => {});
    },
    [rawAppConfig, effectiveAppConfig, updateAppConfig, initialState, state]
  );

  const {
    unhandledCauses: rawUnhandledCauses,
    otherError,
    value: formContextValue,
  } = useValidationError(updateAppConfigError);

  // TODO: refine this error, include more info for distinguishing
  // general validation error
  const errorRules: GenericErrorHandlingRule[] = useMemo(
    () => [
      {
        reason: "ValidationFailed",
        kind: "general",
        jsonPointer: /\/authentication\/identities\/[0-9]+/,
        errorMessageID:
          "AuthenticationAuthenticator.error.no-primary-authenticator",
      },
    ],
    []
  );

  const { errorMessage, unhandledCauses } = useGenericError(
    otherError,
    rawUnhandledCauses,
    errorRules
  );

  return (
    <FormContext.Provider value={formContextValue}>
      <form className={styles.root} onSubmit={onFormSubmit}>
        <NavigationBlockerDialog blockNavigation={isFormModified} />
        <ModifiedIndicatorPortal
          resetForm={resetForm}
          isModified={isFormModified}
        />
        <ErrorDialog error={null} rules={[]} errorMessage={errorMessage} />
        <ShowUnhandledValidationErrorCause causes={unhandledCauses} />
        <div
          className={styles.widget}
          style={{ boxShadow: DefaultEffects.elevation4 }}
        >
          <Text as="h2" className={styles.widgetHeader}>
            <FormattedMessage id="AuthenticationAuthenticator.widgetHeader.primary" />
          </Text>
          <DetailsListWithOrdering
            items={primaryAuthenticators}
            columns={authenticatorColumns}
            onRenderItemColumn={renderPrimaryItemColumn}
            onSwapClicked={onPrimarySwapClicked}
            selectionMode={SelectionMode.none}
            renderAriaLabel={renderPrimaryAriaLabel}
          />
        </div>

        <div
          className={styles.widget}
          style={{ boxShadow: DefaultEffects.elevation4 }}
        >
          <Text as="h2" className={styles.widgetHeader}>
            <FormattedMessage id="AuthenticationAuthenticator.widgetHeader.secondary" />
          </Text>
          <DetailsListWithOrdering
            items={secondaryAuthenticators}
            columns={authenticatorColumns}
            onRenderItemColumn={renderSecondaryItemColumn}
            onSwapClicked={onSecondarySwapClicked}
            selectionMode={SelectionMode.none}
            renderAriaLabel={renderSecondaryAriaLabel}
          />
        </div>

        <section className={styles.policy}>
          <Text className={styles.policyHeader} as="h2">
            <FormattedMessage id="AuthenticationAuthenticator.policy.title" />
          </Text>
          <Dropdown
            className={styles.requireMFADropdown}
            label={renderToString(
              "AuthenticationAuthenticator.policy.require-mfa"
            )}
            options={requireMFAOptions}
            selectedKey={secondaryAuthenticationMode}
            onChange={onRequireMFAOptionChange}
          />
          <FormTextField
            jsonPointer="/authentication/recovery_code/count"
            parentJSONPointer="/authentication/recovery_code"
            fieldName="count"
            fieldNameMessageID="AuthenticationAuthenticator.policy.recovery-code-number"
            className={styles.recoveryCodeNumber}
            value={recoveryCodeNumber}
            onChange={onRecoveryCodeNumberChange}
          />
          <Toggle
            className={styles.allowRetrieveRecoveryCode}
            inlineLabel={true}
            label={
              <FormattedMessage id="AuthenticationAuthenticator.policy.allow-retrieve-recovery-code" />
            }
            checked={allowRetrieveRecoveryCode}
            onChange={onAllowRetrieveRecoveryCodeChange}
          />
        </section>

        <ButtonWithLoading
          type="submit"
          className={styles.saveButton}
          disabled={!isFormModified}
          loading={updatingAppConfig}
          labelId="save"
          loadingLabelId="saving"
        />
      </form>
    </FormContext.Provider>
  );
};

export default AuthenticationAuthenticatorSettings;
