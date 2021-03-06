import { LanguageTag, ResourceUpdate, Resource } from "../../util/resource";

export interface GenerateUpdatesResult {
  isModified: boolean;
  additions: ResourceUpdate[];
  invalidAdditionLocales: LanguageTag[];
  editions: ResourceUpdate[];
  invalidEditionLocales: LanguageTag[];
  deletions: ResourceUpdate[];
}

// eslint-disable-next-line complexity
export function generateUpdates(
  initialTemplateLocales: LanguageTag[],
  initialTemplates: Resource[],
  templateLocales: LanguageTag[],
  templates: Resource[]
): GenerateUpdatesResult {
  // We have 3 kinds of updates
  // 1. Addition
  // 2. Edition
  // 3. Deletion

  // Addition: present in templateLocales but absent in initialTemplateLocales
  const additionLocales: LanguageTag[] = [];
  for (const locale of templateLocales) {
    const idx = initialTemplateLocales.indexOf(locale);
    if (idx < 0) {
      additionLocales.push(locale);
    }
  }
  // It is valid iff there is at least 1 template with non-empty value.
  const invalidAdditionLocales: LanguageTag[] = [];
  const additions: ResourceUpdate[] = [];
  for (const locale of additionLocales) {
    let valid = false;
    for (const template of templates) {
      if (template.specifier.locale === locale && template.value !== "") {
        valid = true;
        additions.push({
          ...template,
        });
      }
    }
    if (!valid) {
      invalidAdditionLocales.push(locale);
    }
  }

  // Edition: present in both templateLocales and initialTemplateLocales
  const editionLocales: LanguageTag[] = [];
  for (const locale of templateLocales) {
    const idx = initialTemplateLocales.indexOf(locale);
    if (idx >= 0) {
      editionLocales.push(locale);
    }
  }
  // It is valid iff there is at least 1 template with non-empty value.
  const invalidEditionLocales: LanguageTag[] = [];
  const editions: ResourceUpdate[] = [];
  for (const locale of editionLocales) {
    let valid = false;

    // Handle resource being modified.
    for (const template of templates) {
      if (template.specifier.locale === locale) {
        if (template.value !== "") {
          valid = true;
        }

        for (const oldTemplate of initialTemplates) {
          if (
            oldTemplate.specifier.locale === template.specifier.locale &&
            oldTemplate.specifier.def === template.specifier.def
          ) {
            if (oldTemplate.value !== template.value) {
              editions.push({
                ...template,
                value: template.value === "" ? null : template.value,
              });
            }
          }
        }
      }
    }

    // Handle resource being added.
    for (const template of templates) {
      if (template.specifier.locale === locale) {
        let found = false;
        for (const oldTemplate of initialTemplates) {
          if (
            template.specifier.locale === oldTemplate.specifier.locale &&
            template.specifier.def === oldTemplate.specifier.def
          ) {
            found = true;
          }
        }
        if (!found) {
          editions.push({
            ...template,
            value: template.value === "" ? null : template.value,
          });
        }
      }
    }

    // Handle resource being removed.
    for (const oldTemplate of initialTemplates) {
      if (oldTemplate.specifier.locale === locale) {
        let found = false;
        for (const template of templates) {
          if (
            oldTemplate.specifier.locale === template.specifier.locale &&
            oldTemplate.specifier.def === template.specifier.def
          ) {
            found = true;
          }
        }
        if (!found) {
          editions.push({
            ...oldTemplate,
            value: null,
          });
        }
      }
    }

    if (!valid) {
      invalidEditionLocales.push(locale);
    }
  }

  // Deletion: present in initialTemplateLocales but absent in templateLocales
  const deletionLocales: LanguageTag[] = [];
  for (const locale of initialTemplateLocales) {
    const idx = templateLocales.indexOf(locale);
    if (idx < 0) {
      deletionLocales.push(locale);
    }
  }
  const deletions: ResourceUpdate[] = [];
  for (const locale of deletionLocales) {
    for (const template of initialTemplates) {
      if (template.specifier.locale === locale) {
        deletions.push({
          ...template,
          value: null,
        });
      }
    }
  }

  const isModified =
    additions.length > 0 ||
    invalidAdditionLocales.length > 0 ||
    editions.length > 0 ||
    invalidEditionLocales.length > 0 ||
    deletions.length > 0;

  return {
    isModified,
    additions,
    invalidAdditionLocales,
    editions,
    invalidEditionLocales,
    deletions,
  };
}
