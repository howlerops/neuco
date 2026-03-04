ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_framework_check;
ALTER TABLE projects ADD CONSTRAINT projects_framework_check
    CHECK (framework IN ('react', 'nextjs', 'vue', 'svelte', 'sveltekit', 'angular', 'solid', 'nuxt', 'other'));

ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_styling_check;
ALTER TABLE projects ADD CONSTRAINT projects_styling_check
    CHECK (styling IN ('tailwind', 'css_modules', 'styled_components', 'vanilla', 'sass', 'plain_css'));
