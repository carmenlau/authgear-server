import React, { useCallback, useMemo } from "react";
import { FormattedMessage } from "@oursky/react-messageformat";
import { Toggle } from "@fluentui/react";
import cn from "classnames";
import produce from "immer";
import { clearEmptyObject } from "../../util/misc";
import { PortalAPIAppConfig } from "../../types";
import {
  AppConfigFormModel,
  useAppConfigForm,
} from "../../hook/useAppConfigForm";
import { useParams } from "react-router-dom";
import ShowLoading from "../../ShowLoading";
import ShowError from "../../ShowError";
import ScreenContent from "../../ScreenContent";
import ScreenTitle from "../../ScreenTitle";
import ScreenDescription from "../../ScreenDescription";
import Widget from "../../Widget";
import WidgetTitle from "../../WidgetTitle";
import FormContainer from "../../FormContainer";
import styles from "./PasswordlessConfigurationScreen.module.scss";

const oobotpAuthenticators = ["oob_otp_email", "oob_otp_sms"];

interface FormState {
  oobotpEmailEnabled: boolean;
  oobotpSMSEnabled: boolean;
  hasOtherAuthenticator: boolean;
  oobotpFirst: boolean;
}

function constructFormState(config: PortalAPIAppConfig): FormState {
  const primaryAuthenticators =
    config.authentication?.primary_authenticators ?? [];

  let oobotpFirst = false;
  if (primaryAuthenticators.length >= 1) {
    oobotpFirst = oobotpAuthenticators.includes(primaryAuthenticators[0]);
  }

  // if there is at least one authenticator other than oob top
  // and at least one oob otp authenticator
  // enable the priority toggle
  let hasOtherAuthenticator = false;
  for (const a of primaryAuthenticators) {
    if (!oobotpAuthenticators.includes(a)) {
      hasOtherAuthenticator = true;
    }
  }

  return {
    oobotpEmailEnabled:
      primaryAuthenticators.includes("oob_otp_email") ?? false,
    oobotpSMSEnabled: primaryAuthenticators.includes("oob_otp_sms") ?? false,
    hasOtherAuthenticator: hasOtherAuthenticator,
    oobotpFirst: oobotpFirst,
  };
}

function constructConfig(
  config: PortalAPIAppConfig,
  initialState: FormState,
  currentState: FormState
): PortalAPIAppConfig {
  return produce(config, (config) => {
    function ensureItems<T>(items: T[], item: T, exists: boolean): T[] {
      let result = [...items];
      if (items.includes(item)) {
        if (!exists) {
          result = result.filter((i) => i !== item);
        }
      } else {
        if (exists) {
          result.push(item);
        }
      }
      return result;
    }

    let primaryAuthenticators =
      config.authentication?.primary_authenticators ?? [];

    if (initialState.oobotpEmailEnabled !== currentState.oobotpEmailEnabled) {
      primaryAuthenticators = ensureItems(
        primaryAuthenticators,
        "oob_otp_email",
        currentState.oobotpEmailEnabled
      );
    }

    if (initialState.oobotpSMSEnabled !== currentState.oobotpSMSEnabled) {
      primaryAuthenticators = ensureItems(
        primaryAuthenticators,
        "oob_otp_sms",
        currentState.oobotpSMSEnabled
      );
    }

    if (initialState.oobotpFirst !== currentState.oobotpFirst) {
      primaryAuthenticators.sort((a, b) => {
        if (currentState.oobotpFirst) {
          return (
            oobotpAuthenticators.indexOf(b) - oobotpAuthenticators.indexOf(a)
          );
        } else {
          return (
            oobotpAuthenticators.indexOf(a) - oobotpAuthenticators.indexOf(b)
          );
        }
      });
    }

    console.log("oobotpAuthenticators1", primaryAuthenticators);

    // console.log("oobotpAuthenticators2", primaryAuthenticators);

    config.authentication ??= {};
    config.authentication.primary_authenticators = primaryAuthenticators;

    clearEmptyObject(config);
  });
}

interface PasswordlessConfigurationScreenContentProps {
  form: AppConfigFormModel<FormState>;
}

const PasswordlessConfigurationScreenContent: React.FC<PasswordlessConfigurationScreenContentProps> =
  function PasswordlessConfigurationScreenContent(props) {
    const { state, setState } = props.form;

    const onOOBOTPEmailEnabledChange = useCallback(
      (_, checked?: boolean) => {
        if (checked == null) {
          return;
        }
        setState((state) => ({
          ...state,
          oobotpEmailEnabled: checked,
        }));
      },
      [setState]
    );

    const onOOBOTPSMSEnabledChange = useCallback(
      (_, checked?: boolean) => {
        if (checked == null) {
          return;
        }
        setState((state) => ({
          ...state,
          oobotpSMSEnabled: checked,
        }));
      },
      [setState]
    );

    const onOOBOTPFirstEnabledChange = useCallback(
      (_, checked?: boolean) => {
        if (checked == null) {
          return;
        }
        setState((state) => ({
          ...state,
          oobotpFirst: checked,
        }));
      },
      [setState]
    );

    const oobotpFirstToggleEnabled = useMemo(() => {
      return (
        state.hasOtherAuthenticator &&
        (state.oobotpEmailEnabled || state.oobotpSMSEnabled)
      );
    }, [
      state.hasOtherAuthenticator,
      state.oobotpEmailEnabled,
      state.oobotpSMSEnabled,
    ]);

    return (
      <ScreenContent>
        <ScreenTitle className={styles.widget}>
          <FormattedMessage id="PasswordlessConfigurationScreen.title" />
        </ScreenTitle>
        <ScreenDescription className={styles.widget}>
          <FormattedMessage id="PasswordlessConfigurationScreen.description" />
        </ScreenDescription>
        <Widget className={styles.widget}>
          <WidgetTitle>
            <FormattedMessage id="PasswordlessConfigurationScreen.authenticators" />
          </WidgetTitle>
          <Toggle
            checked={state.oobotpEmailEnabled}
            inlineLabel={true}
            label={
              <FormattedMessage id="PasswordlessConfigurationScreen.oob-otp-email-enabled.label" />
            }
            onChange={onOOBOTPEmailEnabledChange}
          />
          <Toggle
            checked={state.oobotpSMSEnabled}
            inlineLabel={true}
            label={
              <FormattedMessage id="PasswordlessConfigurationScreen.oob-otp-sms-enabled.label" />
            }
            onChange={onOOBOTPSMSEnabledChange}
          />
        </Widget>
        <Widget
          className={cn(styles.widget, {
            [styles.readOnly]: !oobotpFirstToggleEnabled,
          })}
        >
          <WidgetTitle>
            <FormattedMessage id="PasswordlessConfigurationScreen.priority" />
          </WidgetTitle>
          <Toggle
            checked={oobotpFirstToggleEnabled && state.oobotpFirst}
            inlineLabel={true}
            label={
              <FormattedMessage id="PasswordlessConfigurationScreen.passwordless-first.label" />
            }
            onChange={onOOBOTPFirstEnabledChange}
          />
        </Widget>
      </ScreenContent>
    );
  };

const PasswordlessConfigurationScreen: React.FC =
  function PasswordlessConfigurationScreen() {
    const { appID } = useParams();
    const form = useAppConfigForm(appID, constructFormState, constructConfig);

    if (form.isLoading) {
      return <ShowLoading />;
    }

    if (form.loadError) {
      return <ShowError error={form.loadError} onRetry={form.reload} />;
    }

    return (
      <FormContainer form={form}>
        <PasswordlessConfigurationScreenContent form={form} />
      </FormContainer>
    );
  };

export default PasswordlessConfigurationScreen;
