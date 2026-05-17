# KoiMoe Diary Visual And Atmosphere Guide

This file is the source of truth for visual design, atmosphere, and public-facing copy direction in the Sakurairo Go project. When future UI or content decisions are ambiguous, prefer this guide over generic blog, dashboard, or product-site conventions.

## Core Identity

- Primary title: `KoiMoe Diary`
- Primary subtitle: `恋と萌えの小さな場所`
- Public mood: a small personal place for heartbeats, cute things, two-dimensional/anime aesthetics, and gentle diary-like records.
- Visual goal: preserve the recognizable Sakurairo WordPress theme atmosphere while making it feel native to a lightweight GoFrame blog.
- Product posture: this is currently a personal site for the user, so visual fidelity, emotional tone, and pleasant reading/authoring experience take priority over enterprise admin patterns or defensive multi-user hardening.

## Reference Sources

Use these as primary visual references:

- WordPress theme source: `D:\codex\Sakurairo-1.20.10`
- Main theme CSS: `D:\codex\Sakurairo-1.20.10\style.css`
- Comment form reference: `D:\codex\Sakurairo-1.20.10\comments.php`
- Post list reference: `D:\codex\Sakurairo-1.20.10\tpl\content-thumb.php`
- Single post reference: `D:\codex\Sakurairo-1.20.10\tpl\content-single.php`
- Footer, search, floating controls: `D:\codex\Sakurairo-1.20.10\footer.php`
- Local running WordPress reference while available: `http://localhost:8881/`

Some reference images/CDN assets may fail to load in the old local WordPress site. Analyze layout, spacing, interaction, and intended atmosphere even when individual assets are broken.

## Atmosphere

The site should feel:

- Soft, pale, warm, and slightly dreamy.
- Personal and intimate, like a small diary rather than a publication platform.
- Anime-adjacent and cute without becoming loud or toy-like.
- Airy, with large breathing room and low-contrast typography.
- Sakura-like: pink accents, translucent white surfaces, gentle shadows, quiet motion.

Avoid:

- Generic SaaS/dashboard styling on public pages.
- Heavy black borders, dark chrome, dense utility layouts, and corporate copy.
- Marketing hero sections that feel unrelated to the actual blog.
- Overly technical public wording about Go, PHP, WordPress migration, or implementation details.

## Color And Surfaces

Primary Sakurairo pink:

- `#FB98C0`
- Use for input borders, focus states, soft badges, floating labels, comment accents, and delicate interactive emphasis.

Supporting warm accent:

- `#FE9600`
- Use sparingly for hover details, read-more affordances, or secondary warmth.
- Do not use it as the main form border or dominant brand color.

Surface rules:

- Use translucent white and pale pink-white panels.
- Prefer thin white or pink-tinted borders.
- Use soft, broad shadows instead of hard outlines.
- Cards usually stay around `8px` radius.
- Comment fields in the original theme use softer `15px` rounded corners.
- Black or neutral gray form borders are usually wrong for this theme, especially in comments and replies.

## Home Page

The homepage first impression should be an immersive full-screen hero, not an ordinary post index.

Key elements:

- Large atmospheric background image.
- Minimal top navigation, light in weight, with pink accents.
- Central translucent intro panel with soft text.
- Avatar/social affordances below or near the intro.
- A gentle down-scroll affordance.
- Feature/focus areas that feel airy and decorative.
- Post list cards that use soft shadows and large spacing.

The WordPress reference uses large split post cards:

- Image on the left.
- Text/content on the right.
- Small date/category metadata.
- Pink date pill, around `#FFEEEB`.
- Three-dot decorative feeling.
- Quiet read-more/icon affordance rather than a heavy button.

## Public Navigation

Navigation should be light:

- Minimal link set.
- Thin, airy header.
- No heavy nav bar.
- Search/user icons can be visual affordances.
- Mobile should keep the immersive feeling rather than collapsing into a purely utilitarian menu.

## Search

Search should feel like a Sakurairo overlay:

- Full-screen translucent veil.
- Large rounded search input.
- Centered composition.
- Optional decorative character/image treatment on the side.
- Search should feel atmospheric, not like a plain form page.

## Article Pages

Single post pages should be calm and centered.

Design rules:

- Use a soft cover image band when available.
- Center the title with quiet metadata.
- Use pink-tinted metadata pills for time, views, and comments.
- Keep article content in a translucent paper-like container.
- Preserve generous reading width and line height.
- Style images with gentle radius and soft shadow.
- Style blockquotes with pale pink background and pink left accent.
- Style code blocks clearly, but keep them integrated with the soft page.
- Include a license/attribution area that feels like an information strip, not a legal block.
- Include author card and comment anchor.
- Include previous/next post navigation as part of the reading flow.

## Comments

The comment area is expressive and playful in the original theme. It should not feel like a generic enterprise contact form.

Theme source observations:

- Original form id is `commentform`.
- Submit copy is `BiuBiuBiu~`.
- Textarea class behavior is similar to `.commentbody`.
- Placeholder: `You are a surprise that I will only meet once in my life ...`
- The theme includes playful emoji panel toggles like `Click me OωO`.
- Field flow is closer to Nickname / email / Site than generic Name / Email / Website.

Implementation rules:

- Comment textarea and inputs use `#FB98C0` borders.
- Comment fields use about `15px` radius.
- Focus state uses pink glow and soft white background.
- Comment cards use pink-tinted borders and translucent white backgrounds.
- Avatars can use pink gradients or sakura colors.
- Reply/comment affordances should stay playful and gentle.

## Admin Pages

Admin screens should still feel connected to the public site.

Rules:

- Avoid generic dark dashboards.
- Use the same sakura pink, pale translucent surfaces, and quiet typography.
- Login can be more immersive and decorative than later admin screens.
- The current login direction uses a custom anime train background with a semi-transparent pink-white glass panel.
- Editing workflows should be usable first, but still visually soft.

## Floating Controls

The original Sakurairo theme includes floating controls that contribute to mood:

- Back-to-top.
- Search/user affordances.
- Style/skin menu.
- Font controls.

These are part of the theme atmosphere, not just utilities. Future implementations should treat them as visual features and keep them light, icon-oriented, and unobtrusive.

## Copywriting

Public copy should follow `KoiMoe Diary` and `恋と萌えの小さな場所`.

Prefer:

- Light, poetic, slightly intimate wording.
- Words that suggest memory, small moments, heartbeats, cute things, diary fragments, gentle rooms, soft places.
- English title plus Japanese subtitle when it helps the mood.

Avoid:

- Public copy that sounds like a framework demo.
- Phrases centered on GoFrame, WordPress migration, PHP replacement, or implementation details.
- Over-explaining how the site works in visible UI.

Good baseline:

- Title: `KoiMoe Diary`
- Subtitle: `恋と萌えの小さな場所`
- Notice style: `A soft diary for tiny heartbeats, cute things, and everyday fragments.`

## Implementation Priority

When choosing between competing tasks:

1. Preserve or improve Sakurairo atmosphere.
2. Make the page more pleasant to read, browse, or write.
3. Keep useful blog behavior working.
4. Add security or hardening only when it is cheap and does not damage the experience.

This does not mean ignoring correctness. It means the main product shape is a personal atmospheric blog, not a hardened multi-user platform.

## Current Visual Milestones

- Home page atmosphere polish: intro area, notice styling, feature depth, glassy post cards, sidebar surfaces.
- Comment form alignment: pink borders, 15px rounded fields, playful placeholder, `BiuBiuBiu~`.
- Admin login: custom anime train background with translucent glass panel.
- Single post page: cover image band, translucent article paper, metadata pills, content styling, author card, adjacent post navigation.
- Text direction: `KoiMoe Diary` / `恋と萌えの小さな場所`.

## Future Visual Punch List

- Make home post cards closer to the original split `post-list-thumb` design.
- Strengthen full-screen modal search.
- Add mobile navigation that keeps the immersive hero mood.
- Improve archive/search/category/tag pages with original-theme-like headers and empty states.
- Add skin/style and font controls.
- Add optional sakura effects and richer hero/background settings.
- Polish admin editor and settings pages so they feel like the same site, not a generic dashboard.
