import React, { useContext, useCallback, useState, useMemo } from "react";
import { useParams } from "react-router-dom";
import { produce } from "immer";
import deepEqual from "deep-equal";
import { FormattedMessage, Context } from "@oursky/react-messageformat";
import { Text, Toggle, Dropdown, IDropdownOption } from "@fluentui/react";

import { useAppConfigQuery } from "./query/appConfigQuery";
import { useUpdateAppConfigMutation } from "./mutations/updateAppConfigMutation";
import {
  PortalAPIAppConfig,
  PromotionConflictBehaviour,
  promotionConflictBehaviours,
  isPromotionConflictBehaviour,
} from "../../types";
import { clearEmptyObject } from "../../util/misc";
import ShowLoading from "../../ShowLoading";
import ShowError from "../../ShowError";
import ErrorDialog from "../../error/ErrorDialog";
import ButtonWithLoading from "../../ButtonWithLoading";
import NavigationBlockerDialog from "../../NavigationBlockerDialog";
import {
  ModifiedIndicatorPortal,
  ModifiedIndicatorWrapper,
} from "../../ModifiedIndicatorPortal";

import styles from "./AnonymousUsersConfigurationScreen.module.scss";

interface AnonymousUsersConfigurationScreenState {
  enabled: boolean;
  promotionConflictBehaviour: PromotionConflictBehaviour;
}

interface AnonymousUsersConfigurationProps {
  effectiveAppConfig: PortalAPIAppConfig | null;
  rawAppConfig: PortalAPIAppConfig | null;
  resetForm: () => void;
}

const DEFAULT_CONFLICT_BEHAVIOUR: PromotionConflictBehaviour = "error";

const conflictBehaviourMessageId: Record<PromotionConflictBehaviour, string> = {
  login: "AnonymousIdentityConflictBehaviour.login",
  error: "AnonymousIdentityConflictBehaviour.error",
};

function constructStateFromAppConfig(
  appConfig: PortalAPIAppConfig | null
): AnonymousUsersConfigurationScreenState {
  if (appConfig == null) {
    return {
      enabled: false,
      promotionConflictBehaviour: DEFAULT_CONFLICT_BEHAVIOUR,
    };
  }
  const anonymousUserEnabled =
    appConfig.authentication?.identities?.find(
      (identity) => identity === "anonymous"
    ) != null;
  const promotionConflictBehaviour =
    appConfig.identity?.on_conflict?.promotion ?? DEFAULT_CONFLICT_BEHAVIOUR;
  return {
    enabled: anonymousUserEnabled,
    promotionConflictBehaviour,
  };
}

function constructNewAppConfigFromState(
  state: AnonymousUsersConfigurationScreenState,
  initialState: AnonymousUsersConfigurationScreenState,
  rawAppConfig: PortalAPIAppConfig,
  effectiveAppConfig: PortalAPIAppConfig
) {
  return produce(rawAppConfig, (draftConfig) => {
    draftConfig.identity = draftConfig.identity ?? {};
    draftConfig.identity.on_conflict = draftConfig.identity.on_conflict ?? {};
    const onConflict = draftConfig.identity.on_conflict;

    draftConfig.authentication = draftConfig.authentication ?? {};
    // if raw config does not contain authentication.identities, effective
    // config is ["oauth", "login_id"], so effective config is used to
    // avoid enabling anonymous user will remove "oauth" and "login_id"
    // from effective config of authentication.identities
    draftConfig.authentication.identities =
      effectiveAppConfig.authentication?.identities ?? [];
    const { authentication } = draftConfig;
    const authenticationIdentitiesSet = new Set(authentication.identities);

    const enabledStateChanged = state.enabled !== initialState.enabled;
    const behaviourStateChanged =
      state.promotionConflictBehaviour !==
      initialState.promotionConflictBehaviour;

    if (state.enabled) {
      if (enabledStateChanged) {
        authenticationIdentitiesSet.add("anonymous");
        authentication.identities = Array.from(authenticationIdentitiesSet);
      }
      if (behaviourStateChanged) {
        onConflict.promotion = state.promotionConflictBehaviour;
      }
    } else {
      if (enabledStateChanged) {
        authenticationIdentitiesSet.delete("anonymous");
        authentication.identities = Array.from(authenticationIdentitiesSet);
      }
    }

    clearEmptyObject(draftConfig);
  });
}

function constructConflictBehaviourOptions(
  state: AnonymousUsersConfigurationScreenState,
  renderToString: (messageId: string) => string
): IDropdownOption[] {
  return promotionConflictBehaviours.map((behaviour) => {
    const selectedBehaviour = state.promotionConflictBehaviour;
    return {
      key: behaviour,
      text: renderToString(conflictBehaviourMessageId[behaviour]),
      isSelected: selectedBehaviour === behaviour,
    };
  });
}

const AnonymousUsersConfiguration: React.FC<AnonymousUsersConfigurationProps> = function AnonymousUsersConfiguration(
  props: AnonymousUsersConfigurationProps
) {
  const { effectiveAppConfig, rawAppConfig, resetForm } = props;

  const { renderToString } = useContext(Context);
  const { appID } = useParams();

  const {
    loading: updatingAppConfig,
    error: updateAppConfigError,
    updateAppConfig,
  } = useUpdateAppConfigMutation(appID);

  const initialState = useMemo(
    () => constructStateFromAppConfig(effectiveAppConfig),
    [effectiveAppConfig]
  );

  const [state, setState] = useState(initialState);
  const conflictBehaviourOptions = useMemo(() => {
    return constructConflictBehaviourOptions(state, renderToString);
  }, [state, renderToString]);

  const isFormModified = useMemo(() => {
    return !deepEqual(initialState, state, { strict: true });
  }, [initialState, state]);

  const onSwitchToggled = useCallback(
    (_event, checked?: boolean) => {
      if (checked == null) {
        return;
      }
      setState({
        ...state,
        enabled: checked,
      });
    },
    [state]
  );

  const onConflictOptionChange = useCallback(
    (_event, option?: IDropdownOption) => {
      if (option != null && isPromotionConflictBehaviour(option.key)) {
        setState({
          ...state,
          promotionConflictBehaviour: option.key,
        });
      }
    },
    [state]
  );

  const onSaveClicked = useCallback(() => {
    if (rawAppConfig == null || effectiveAppConfig == null) {
      return;
    }
    const newAppConfig = constructNewAppConfigFromState(
      state,
      initialState,
      rawAppConfig,
      effectiveAppConfig
    );

    updateAppConfig(newAppConfig).catch(() => {});
  }, [updateAppConfig, rawAppConfig, effectiveAppConfig, initialState, state]);

  return (
    <section className={styles.screenContent}>
      <ErrorDialog error={updateAppConfigError} rules={[]} />
      <NavigationBlockerDialog blockNavigation={isFormModified} />
      <ModifiedIndicatorPortal
        resetForm={resetForm}
        isModified={isFormModified}
      />
      <Toggle
        className={styles.enableToggle}
        checked={state.enabled}
        onChange={onSwitchToggled}
        label={renderToString("AnonymousUsersConfigurationScreen.enable.label")}
        inlineLabel={true}
      />
      <Dropdown
        className={styles.conflictDropdown}
        label={renderToString(
          "AnonymousUsersConfigurationScreen.conflict-droplist.label"
        )}
        disabled={!state.enabled}
        options={conflictBehaviourOptions}
        selectedKey={state.promotionConflictBehaviour}
        onChange={onConflictOptionChange}
      />
      <ButtonWithLoading
        disabled={!isFormModified}
        className={styles.saveButton}
        onClick={onSaveClicked}
        loading={updatingAppConfig}
        labelId="save"
        loadingLabelId="saving"
      />
    </section>
  );
};

const AnonymousUserConfigurationScreen: React.FC = function AnonymousUserConfigurationScreen() {
  const { appID } = useParams();
  const {
    effectiveAppConfig,
    rawAppConfig,
    loading,
    error,
    refetch,
  } = useAppConfigQuery(appID);

  const [remountIdentifier, setRemountIdentifier] = useState(0);
  const resetForm = useCallback(() => {
    setRemountIdentifier((prev) => prev + 1);
  }, []);

  if (loading) {
    return <ShowLoading />;
  }

  if (error != null) {
    return <ShowError error={error} onRetry={refetch} />;
  }

  return (
    <main className={styles.root}>
      <ModifiedIndicatorWrapper className={styles.wrapper}>
        <Text as="h1" className={styles.title}>
          <FormattedMessage id="AnonymousUsersConfigurationScreen.title" />
        </Text>
        <AnonymousUsersConfiguration
          key={remountIdentifier}
          effectiveAppConfig={effectiveAppConfig}
          rawAppConfig={rawAppConfig}
          resetForm={resetForm}
        />
      </ModifiedIndicatorWrapper>
    </main>
  );
};

export default AnonymousUserConfigurationScreen;
