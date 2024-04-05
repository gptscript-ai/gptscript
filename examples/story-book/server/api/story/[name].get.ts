import fs from 'fs'

type Pages = Record<string, Page>;
type Page = {
    image_path: string;
    content: string;
}

export default defineEventHandler(async (event) => {
    try {
        let name = getRouterParam(event, 'name');
        if (!name) {
            throw createError({
                statusCode: 400,
                statusMessage: 'name is required'
            });
        }

        name = decodeURIComponent(name);

        const files = await fs.promises.readdir(`public/stories/${name}`)

        const pages: Pages = {};
        for (const file of files) {
            if (!file.endsWith('.txt')) continue
            const page = await fs.promises.readFile(`public/stories/${name}/${file}`, 'utf-8')
            pages[ file.replace('.txt', '').replace('page', '')] = {
                image_path: `/stories/${name}/${file.replace('.txt', '.png')}`,
                content: page
            }
        }

        return pages
    } catch (error) {
        // if the error is a 404 error, we can throw it directly
        if ((error as any).code === 'ENOENT') {
            throw createError({
                statusCode:    404,
                statusMessage: 'story found',
            })
        }
        throw createError({
            statusCode:    500,
            statusMessage: `error fetching story: ${error}`,
        })
    }
})